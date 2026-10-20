package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mldl-exam-trainer/internal/auth"
	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/db"
	webserver "mldl-exam-trainer/internal/http"
	"mldl-exam-trainer/web"
)

var address = "0.0.0.0:8080"

func main() {
	if configuredAddress := os.Getenv("EXAM_TRAINER_ADDR"); configuredAddress != "" {
		address = configuredAddress
	}
	catalog, err := content.Load("content")
	if err != nil {
		log.Fatalf("Lerninhalte konnten nicht geladen werden: %v", err)
	}
	for _, warning := range content.CheckSourceAvailability(catalog, ".") {
		log.Printf("Warnung: %s", warning)
	}

	databasePath := os.Getenv("APP_DB_PATH")
	if databasePath == "" {
		databasePath = "/app/data/exam.db"
	}
	store, err := db.Open(databasePath)
	if err != nil {
		log.Fatalf("Datenbank konnte nicht geöffnet werden: %v", err)
	}
	defer store.Close()
	login := auth.New(store)
	if err := login.ConfigureTrustedProxies(os.Getenv("APP_TRUSTED_PROXY_CIDRS")); err != nil {
		log.Fatalf("Proxy-Konfiguration ungültig: %v", err)
	}
	if err := login.BootstrapAdmin(os.Getenv("ADMIN_USERNAME"), os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Fatalf("Admin-Bootstrap fehlgeschlagen: %v", err)
	}
	// The bootstrap secret is never needed after this point in the process.
	_ = os.Unsetenv("ADMIN_PASSWORD")
	_ = os.Unsetenv("ADMIN_USERNAME")
	if err := login.DeleteExpiredSessions(); err != nil {
		log.Fatalf("Abgelaufene Sessions konnten nicht bereinigt werden: %v", err)
	}
	if err := store.SyncCatalog(catalog); err != nil {
		log.Fatalf("Lerninhalte konnten nicht synchronisiert werden: %v", err)
	}
	if err := store.BackfillReviews(catalog); err != nil {
		log.Fatalf("Bestehende Reviews konnten nicht migriert werden: %v", err)
	}
	if err := store.RefreshTopics(catalog); err != nil {
		log.Fatalf("Topic-Fortschritt konnte nicht aktualisiert werden: %v", err)
	}
	if err := store.CheckIntegrity(catalog); err != nil {
		log.Fatalf("Datenintegrität konnte nicht bestätigt werden: %v", err)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           webserver.NewHandlerWithAuth(store, catalog, ".", web.Files, login),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 * 1024,
	}

	go func() {
		log.Printf("ML/DL Exam Trainer läuft auf http://%s", address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP-Server fehlgeschlagen: %v", err)
		}
	}()
	cleanupContext, stopCleanup := context.WithCancel(context.Background())
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupContext.Done():
				return
			case <-ticker.C:
				if err := login.DeleteExpiredSessions(); err != nil {
					log.Printf("Session-Bereinigung fehlgeschlagen: %v", err)
				}
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	stopCleanup()
	<-cleanupDone

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server konnte nicht sauber beendet werden: %v", err)
	}
}
