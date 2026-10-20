package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"mldl-exam-trainer/internal/content"
	"mldl-exam-trainer/internal/db"
)

func TestPreviewIsLocalAndDoesNotCreateUsersOrSessions(t *testing.T) {
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	handler := previewHandler(store, &content.Catalog{}, fstest.MapFS{"index.html": {Data: []byte("preview")}})
	for _, tc := range []struct {
		name, path, host, peer, origin, site string
		want                                 int
	}{
		{"status", "/api/auth/status", previewAddress, "127.0.0.1:1234", "", "", 200},
		{"catalog", "/api/catalog", previewAddress, "127.0.0.1:1234", "", "same-origin", 200},
		{"admin denied", "/api/admin/users", previewAddress, "127.0.0.1:1234", "", "", 403},
		{"logout denied", "/api/auth/logout", previewAddress, "127.0.0.1:1234", "", "", 403},
		{"password denied", "/api/auth/password", previewAddress, "127.0.0.1:1234", "", "", 403},
		{"remote peer", "/api/auth/status", previewAddress, "192.0.2.1:1234", "", "", 403},
		{"invalid peer", "/api/auth/status", previewAddress, "garbage", "", "", 403},
		{"DNS rebinding", "/api/catalog", "evil.example:18087", "127.0.0.1:1234", "", "", 403},
		{"cross-site origin", "/api/catalog", previewAddress, "127.0.0.1:1234", "https://evil.example", "", 403},
		{"null origin", "/api/catalog", previewAddress, "127.0.0.1:1234", "null", "", 403},
		{"cross-site fetch", "/api/catalog", previewAddress, "127.0.0.1:1234", "", "cross-site", 403},
		{"same-origin", "/api/catalog", previewAddress, "127.0.0.1:1234", "http://" + previewAddress, "same-origin", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://"+tc.host+tc.path, nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if tc.name == "status" && !strings.Contains(w.Body.String(), `"authenticated":true`) {
				t.Fatal("preview does not skip the login screen")
			}
			if len(w.Result().Cookies()) != 0 {
				t.Fatal("preview unexpectedly created auth cookies")
			}
		})
	}
	for _, table := range []string{"users", "sessions"} {
		var count int
		query := "SELECT COUNT(*) FROM users"
		if table == "sessions" {
			query = "SELECT COUNT(*) FROM sessions"
		}
		if err := store.DB.QueryRow(query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s changed: %d, %v", table, count, err)
		}
	}
}
