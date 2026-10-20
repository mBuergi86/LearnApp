package auth

import (
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mldl-exam-trainer/internal/db"
)

func TestProxyHeadersRequireExplicitTrustedPeer(t *testing.T) {
	for _, tc := range []struct{ name, config, peer, header, want string }{
		{"direct spoof", "", "203.0.113.1:1234", "198.51.100.2", "203.0.113.1"},
		{"loopback not trusted by default", "", "127.0.0.1:1234", "198.51.100.2", "127.0.0.1"},
		{"trusted peer", "10.0.0.0/24", "10.0.0.2:1234", "198.51.100.2", "198.51.100.2"},
		{"forged left side", "10.0.0.0/24", "10.0.0.2:1234", "192.0.2.99, 198.51.100.2", "198.51.100.2"},
		{"two trusted hops", "10.0.0.0/24", "10.0.0.2:1234", "192.0.2.99, 198.51.100.2, 10.0.0.3", "198.51.100.2"},
		{"invalid chain falls back", "10.0.0.0/24", "10.0.0.2:1234", "bad-IP", "10.0.0.2"},
		{"IPv6 direct", "", "[2001:db8::1]:1234", "198.51.100.2", "2001:db8::1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := New(nil)
			if err := s.ConfigureTrustedProxies(tc.config); err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "http://example.test/api/auth/login", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Forwarded-For", tc.header)
			if got := s.clientIP(r); got != tc.want {
				t.Fatalf("IP=%q, want %q", got, tc.want)
			}
		})
	}
	for _, invalid := range []string{"not-cidr", "0.0.0.0/0", "::/0"} {
		if err := New(nil).ConfigureTrustedProxies(invalid); err == nil {
			t.Fatalf("unsafe configuration accepted: %q", invalid)
		}
	}
}

func TestDummyHashUsesValidArgon2Encoding(t *testing.T) {
	parts := strings.Split(dummyPasswordHash, "$")
	for _, index := range []int{4, 5} {
		value, err := base64.RawStdEncoding.DecodeString(parts[index])
		if err != nil || (index == 4 && len(value) != 16) || (index == 5 && len(value) != 32) {
			t.Fatalf("invalid dummy encoding: field %d", index)
		}
	}
	if verifyPassword(dummyPasswordHash, "definitely not the dummy password") {
		t.Fatal("unexpected dummy match")
	}
}

func TestLoginLimiterEnforcesWindowAndRemainsBounded(t *testing.T) {
	l := newLimiter()
	now := time.Now()
	for i := 0; i < 5; i++ {
		l.fail("client", "user", now)
	}
	if wait := l.retry("client", "other-user", now.Add(2*time.Second)); wait < 58*time.Second {
		t.Fatalf("IP window bypassed: %v", wait)
	}
	if wait := l.retry("another-ip", "user", now.Add(2*time.Second)); wait < 58*time.Second {
		t.Fatalf("username window bypassed: %v", wait)
	}
	if wait := l.retry("client", "user", now.Add(time.Minute)); wait != 0 {
		t.Fatalf("permanent lock: %v", wait)
	}
	for i := 0; i < 6000; i++ {
		l.fail(fmt.Sprint(i), fmt.Sprint(i), now)
	}
	if len(l.entries) > 10000 {
		t.Fatalf("unbounded entries: %d", len(l.entries))
	}
	l.retry("fresh", "fresh", now.Add(11*time.Minute))
	if len(l.entries) != 0 {
		t.Fatalf("expired limiter entries retained: %d", len(l.entries))
	}
}

func TestLoginRejectsExcessArgonWorkBeforeAllocating(t *testing.T) {
	s := New(nil)
	s.passwordSlots <- struct{}{}
	s.passwordSlots <- struct{}{}
	_, err := s.Login(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/auth/login", nil), "unknown", "wrong")
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("excess password work not rejected: %v", err)
	}
}

func TestLogoutDoesNotClaimSuccessOnDatabaseFailure(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/app.db")
	if err != nil {
		t.Fatal(err)
	}
	s := New(store)
	store.Close()
	w := httptest.NewRecorder()
	if err := s.Logout(w, Principal{Session: "session-id"}); err == nil {
		t.Fatal("logout ignored deletion failure")
	}
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("logout cleared cookies despite failed invalidation")
	}
}

func TestExpiredSessionCleanupKeepsValidSessions(t *testing.T) {
	store, err := db.Open(t.TempDir() + "/app.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := New(store)
	if err := s.CreateUser("cleanup-user", "cleanup test long password", RoleUser); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	s.now = func() time.Time { return now }
	for i, name := range []string{"valid", "idle", "absolute"} {
		last := now
		expiry := now.Add(time.Hour)
		if name == "idle" {
			last = now.Add(-31 * time.Minute)
		}
		if name == "absolute" {
			expiry = now.Add(-time.Second)
		}
		_, err = store.DB.Exec(`INSERT INTO sessions(id,session_hash,csrf_hash,user_id,created_at,last_seen_at,expires_at) VALUES(?,?,?,1,?,?,?)`, name, []byte{byte(i)}, []byte{byte(i)}, now.Format(time.RFC3339Nano), last.Format(time.RFC3339Nano), expiry.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteExpiredSessions(); err != nil {
		t.Fatal(err)
	}
	var count int
	var remaining string
	if err := store.DB.QueryRow(`SELECT COUNT(*),id FROM sessions`).Scan(&count, &remaining); err != nil {
		t.Fatal(err)
	}
	if count != 1 || remaining != "valid" {
		t.Fatalf("cleanup left %d sessions, id=%q", count, remaining)
	}
}
