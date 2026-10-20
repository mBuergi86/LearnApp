package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"mldl-exam-trainer/internal/auth"
)

func TestHTTPSHeadersAndOriginRequireTrustedProxy(t *testing.T) {
	login := auth.New(nil)
	if err := login.ConfigureTrustedProxies("10.0.0.0/24"); err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := withSecurityHeaders(withSameOrigin(next, login), login)
	for _, tc := range []struct {
		peer, origin string
		status       int
		hsts         bool
	}{
		{"203.0.113.1:1234", "http://app.test", 200, false},
		{"203.0.113.1:1234", "https://app.test", 403, false},
		{"10.0.0.2:1234", "https://app.test", 200, true},
		{"10.0.0.2:1234", "https://evil.test", 403, true},
	} {
		r := httptest.NewRequest("POST", "http://app.test/api/settings/theme", strings.NewReader(`{}`))
		r.RemoteAddr = tc.peer
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status || (w.Header().Get("Strict-Transport-Security") != "") != tc.hsts {
			t.Fatalf("peer %s origin %s: status %d HSTS %q", tc.peer, tc.origin, w.Code, w.Header().Get("Strict-Transport-Security"))
		}
	}
}

func TestSameOriginLoginBehindUntrustedTLSProxy(t *testing.T) {
	_, store, catalog := testEnvironment(t)
	login := auth.New(store)
	if err := login.CreateUser("learner", "correct horse battery staple", auth.RoleUser); err != nil {
		t.Fatal(err)
	}
	handler := NewHandlerWithAuth(store, catalog, t.TempDir(), fstest.MapFS{"index.html": {Data: []byte("test")}}, login)
	for _, tc := range []struct {
		origin, site string
		status       int
	}{
		{"https://learn.example.com", "same-origin", http.StatusOK},
		{"https://evil.example", "same-origin", http.StatusForbidden},
		{"https://evil.example", "cross-site", http.StatusForbidden},
		{"https://learn.example.com", "", http.StatusForbidden},
	} {
		r := jsonRequest(http.MethodPost, "/api/auth/login", `{"username":"learner","password":"correct horse battery staple"}`)
		r.Host = "learn.example.com"
		r.RemoteAddr = "10.42.0.7:1234"
		r.Header.Set("Origin", tc.origin)
		r.Header.Set("Sec-Fetch-Site", tc.site)
		r.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("origin %s site %q: status %d body %s", tc.origin, tc.site, w.Code, w.Body.String())
		}
		if w.Header().Get("Strict-Transport-Security") != "" {
			t.Fatal("untrusted forwarded proto enabled HSTS")
		}
	}
}
