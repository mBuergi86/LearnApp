package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"mldl-exam-trainer/internal/auth"
	"mldl-exam-trainer/internal/db"
)

func authenticatedHandler(t *testing.T, role auth.Role) (http.Handler, *auth.Service, *db.Store, *http.Cookie, *http.Cookie) {
	t.Helper()
	_, store, catalog := testEnvironment(t)
	service := auth.New(store)
	if err := service.CreateUser("learner", "correct horse battery staple", role); err != nil {
		t.Fatal(err)
	}
	var hash string
	if err := store.DB.QueryRow(`SELECT password_hash FROM users WHERE username=?`, "learner").Scan(&hash); err != nil {
		t.Fatal(err)
	}
	handler := NewHandlerWithAuth(store, catalog, t.TempDir(), fstest.MapFS{"index.html": {Data: []byte("test")}}, service)
	response := httptest.NewRecorder()
	request := jsonRequest(http.MethodPost, "/api/auth/login", `{"username":"learner","password":"correct horse battery staple"}`)
	request.RemoteAddr = "203.0.113.8:1234"
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body.String())
	}
	var sessionCount int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&sessionCount); err != nil || sessionCount != 1 {
		t.Fatalf("server session was not created: count=%d err=%v", sessionCount, err)
	}
	var session, csrf *http.Cookie
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == auth.SessionCookie {
			session = cookie
		}
		if cookie.Name == auth.CSRFCookie {
			csrf = cookie
		}
	}
	if session == nil || csrf == nil {
		t.Fatal("login did not set both cookies")
	}
	return handler, service, store, session, csrf
}

func requestWithSession(method, path string, session, csrf *http.Cookie, body string, csrfHeader bool) *http.Request {
	request := jsonRequest(method, path, body)
	request.AddCookie(session)
	request.AddCookie(csrf)
	if csrfHeader {
		request.Header.Set(auth.CSRFHeader, csrf.Value)
	}
	return request
}

func TestAuthLoginInvalidUnknownInjectionAndRateLimit(t *testing.T) {
	_, store, catalog := testEnvironment(t)
	service := auth.New(store)
	if err := service.CreateUser("learner", "correct horse battery staple", auth.RoleUser); err != nil {
		t.Fatal(err)
	}
	handler := NewHandlerWithAuth(store, catalog, t.TempDir(), fstest.MapFS{"index.html": {Data: []byte("test")}}, service)
	for _, body := range []string{`{"username":"learner","password":"wrong"}`, `{"username":"nobody","password":"wrong"}`, `{"username":"' OR 1=1 --","password":"wrong"}`} {
		response := httptest.NewRecorder()
		request := jsonRequest(http.MethodPost, "/api/auth/login", body)
		request.RemoteAddr = "198.51.100.3:1"
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), "Benutzername oder Passwort ungültig") {
			t.Fatalf("body=%s status=%d response=%s", body, response.Code, response.Body.String())
		}
	}
	for i := 0; i < 2; i++ {
		response := httptest.NewRecorder()
		request := jsonRequest(http.MethodPost, "/api/auth/login", `{"username":"learner","password":"wrong"}`)
		request.RemoteAddr = "198.51.100.3:1"
		handler.ServeHTTP(response, request)
	}
	limited := httptest.NewRecorder()
	attempt := jsonRequest(http.MethodPost, "/api/auth/login", `{"username":"learner","password":"wrong"}`)
	attempt.RemoteAddr = "198.51.100.3:1"
	handler.ServeHTTP(limited, attempt)
	if limited.Code != http.StatusTooManyRequests || limited.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit status=%d", limited.Code)
	}
}

func TestAuthSessionCSRFLogoutAndCookieFlags(t *testing.T) {
	handler, _, _, session, csrf := authenticatedHandler(t, auth.RoleUser)
	if !session.HttpOnly || !session.Secure || session.SameSite != http.SameSiteLaxMode || session.Path != "/" || session.MaxAge != 1800 {
		t.Fatalf("unsafe session cookie: %+v", *session)
	}
	blocked := httptest.NewRecorder()
	handler.ServeHTTP(blocked, requestWithSession(http.MethodPost, "/api/settings/theme", session, csrf, `{"theme":"dark"}`, false))
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("CSRF attack status=%d", blocked.Code)
	}
	allowed := httptest.NewRecorder()
	handler.ServeHTTP(allowed, requestWithSession(http.MethodPost, "/api/settings/theme", session, csrf, `{"theme":"dark"}`, true))
	if allowed.Code != http.StatusOK {
		t.Fatalf("csrf-valid status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	logout := httptest.NewRecorder()
	handler.ServeHTTP(logout, requestWithSession(http.MethodPost, "/api/auth/logout", session, csrf, `{}`, true))
	if logout.Code != http.StatusOK {
		t.Fatal(logout.Code)
	}
	after := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	request.AddCookie(session)
	handler.ServeHTTP(after, request)
	if after.Code != http.StatusUnauthorized {
		t.Fatalf("logout did not invalidate: %d", after.Code)
	}
}

func TestAuthSessionIdleAbsoluteAndTampering(t *testing.T) {
	handler, _, store, session, _ := authenticatedHandler(t, auth.RoleUser)
	serviceNow := time.Now().UTC().Add(-31 * time.Minute).Format(time.RFC3339Nano)
	if _, err := store.DB.Exec(`UPDATE sessions SET last_seen_at=?`, serviceNow); err != nil {
		t.Fatal(err)
	}
	idle := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	request.AddCookie(session)
	handler.ServeHTTP(idle, request)
	if idle.Code != http.StatusUnauthorized {
		t.Fatalf("idle status=%d", idle.Code)
	}
	// A made-up 256-bit cookie must not become an authenticated session.
	tampered := httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	bad.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: strings.Repeat("a", 43)})
	handler.ServeHTTP(tampered, bad)
	if tampered.Code != http.StatusUnauthorized {
		t.Fatalf("tampered=%d", tampered.Code)
	}
	absoluteHandler, _, absoluteStore, absoluteSession, _ := authenticatedHandler(t, auth.RoleUser)
	if _, err := absoluteStore.DB.Exec(`UPDATE sessions SET expires_at=?`, time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	expired := httptest.NewRecorder()
	expiredRequest := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	expiredRequest.AddCookie(absoluteSession)
	absoluteHandler.ServeHTTP(expired, expiredRequest)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("absolute expiry status=%d", expired.Code)
	}
}

func TestAuthPasswordChangeAndAdminRole(t *testing.T) {
	handler, _, _, session, csrf := authenticatedHandler(t, auth.RoleUser)
	forbidden := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	request.AddCookie(session)
	request.AddCookie(csrf)
	handler.ServeHTTP(forbidden, request)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("normal user admin status=%d", forbidden.Code)
	}
	changed := httptest.NewRecorder()
	handler.ServeHTTP(changed, requestWithSession(http.MethodPost, "/api/auth/password", session, csrf, `{"current_password":"correct horse battery staple","new_password":"another correct battery staple"}`, true))
	if changed.Code != http.StatusOK {
		t.Fatalf("password change status=%d body=%s", changed.Code, changed.Body.String())
	}
	old := httptest.NewRecorder()
	oldRequest := httptest.NewRequest(http.MethodGet, "/api/catalog", nil)
	oldRequest.AddCookie(session)
	handler.ServeHTTP(old, oldRequest)
	if old.Code != http.StatusUnauthorized {
		t.Fatalf("old session remained valid: %d", old.Code)
	}

	adminHandler, _, _, adminSession, _ := authenticatedHandler(t, auth.RoleAdmin)
	allowed := httptest.NewRecorder()
	adminRequest := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	adminRequest.AddCookie(adminSession)
	adminHandler.ServeHTTP(allowed, adminRequest)
	if allowed.Code != http.StatusOK {
		t.Fatalf("admin status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}
