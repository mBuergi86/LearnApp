// Preview is a separate, development-only entrypoint. The container builds
// cmd/server instead, so this passwordless mode cannot be enabled by an env var
// in the production executable.
package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/db"
	webserver "mldl-exam-trainer/internal/http"
	"mldl-exam-trainer/web"
)

const previewAddress = "127.0.0.1:18087"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	catalog, err := content.Load("content")
	if err != nil {
		return err
	}
	// Deliberately ignore APP_DB_PATH, bootstrap secrets and proxy settings.
	// Never open the real learning database for a passwordless preview.
	directory, err := os.MkdirTemp("", "mldl-passwordless-preview-")
	if err != nil {
		return err
	}
	store, err := db.Open(filepath.Join(directory, "exam.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.SyncCatalog(catalog); err != nil {
		return err
	}
	if err := store.RefreshTopics(catalog); err != nil {
		return err
	}
	if err := store.CheckIntegrity(catalog); err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", previewAddress)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           previewHandler(store, catalog, web.Files),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 32 * 1024,
	}
	log.Printf("LOKALE PREVIEW OHNE LOGIN: http://%s", previewAddress)
	log.Printf("Nur temporäre Preview-Daten: %s (bleiben nach Beenden erhalten)", directory)
	log.Print("Produktiver Login bleibt aktiv; zurück mit: go run ./cmd/server")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}

func previewHandler(store *db.Store, catalog *content.Catalog, files fs.FS) http.Handler {
	app := webserver.NewHandler(store, catalog, ".", files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		peer := net.ParseIP(host)
		if err != nil || peer == nil || !peer.IsLoopback() || r.Host != previewAddress {
			http.Error(w, "Preview nur direkt auf 127.0.0.1 erreichbar.", http.StatusForbidden)
			return
		}
		// Strict authority also blocks DNS rebinding. No forwarded host/IP is
		// trusted, and even read requests from cross-site pages are rejected.
		if origin := r.Header.Get("Origin"); (origin != "" && origin != "http://"+previewAddress) || strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			http.Error(w, "Cross-Site-Preview-Anfrage abgelehnt.", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/api/auth/status" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authenticated":true,"local_preview":true}`))
			return
		}
		// These are not meaningful without users/sessions. Do not call the
		// production auth handlers with a nil auth service or grant admin access.
		if strings.HasPrefix(r.URL.Path, "/api/auth/") || strings.HasPrefix(r.URL.Path, "/api/admin/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"Lokale Preview ohne Login: keine Benutzer- oder Sitzungsverwaltung."}`))
			return
		}
		app.ServeHTTP(w, r)
	})
}
