// Package auth implements database-backed browser authentication.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"mldl-exam-trainer/internal/db"
)

const (
	SessionCookie        = "exam_trainer_session"
	CSRFCookie           = "exam_trainer_csrf"
	CSRFHeader           = "X-CSRF-Token"
	idleTimeout          = 30 * time.Minute
	absoluteTTL          = 12 * time.Hour
	minPassword          = 15
	argonMemory   uint32 = 64 * 1024 // KiB, 64 MiB
	argonTime     uint32 = 3
	argonThreads  uint8  = 2
	argonKeyLen   uint32 = 32
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Principal struct {
	ID       int64
	Username string
	Role     Role
	Session  string
	csrfHash [32]byte
}
type Service struct {
	store          *db.Store
	now            func() time.Time
	limits         *limiter
	trustedProxies []netip.Prefix
	passwordSlots  chan struct{}
}

func New(store *db.Store) *Service {
	return &Service{store: store, now: time.Now, limits: newLimiter(), passwordSlots: make(chan struct{}, 2)}
}

func (s *Service) BootstrapAdmin(username, password string) error {
	var users int
	if err := s.store.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		return fmt.Errorf("Benutzer zählen: %w", err)
	}
	if users > 0 {
		return nil
	}
	if strings.TrimSpace(username) == "" || password == "" {
		return errors.New("ADMIN_USERNAME und ADMIN_PASSWORD müssen beim ersten Start gesetzt sein")
	}
	if err := s.CreateUser(username, password, RoleAdmin); err != nil {
		return fmt.Errorf("initialen Admin anlegen: %w", err)
	}
	log.Printf("security event=initial_admin_created username=%q", username)
	return nil
}

func (s *Service) CreateUser(username, password string, role Role) error {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 128 || (role != RoleUser && role != RoleAdmin) {
		return errors.New("ungültiger Benutzer")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	_, err = s.store.DB.Exec(`INSERT INTO users (username, password_hash, role, created_at, updated_at, disabled) VALUES (?, ?, ?, ?, ?, 0)`, username, hash, role, now, now)
	if err != nil {
		return fmt.Errorf("Benutzer speichern: %w", err)
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if len([]rune(password)) < minPassword {
		return "", fmt.Errorf("Passwort muss mindestens %d Zeichen enthalten", minPassword)
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func verifyPassword(encoded, password string) bool {
	var memory, iterations uint32
	var parallelism uint8
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil || memory != argonMemory || iterations != argonTime || parallelism != argonThreads {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) != int(argonKeyLen) {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func (s *Service) Login(w http.ResponseWriter, r *http.Request, username, password string) (Principal, error) {
	keyIP, keyUser := s.clientIP(r), strings.ToLower(strings.TrimSpace(username))
	if retry := s.limits.retry(keyIP, keyUser, s.now()); retry > 0 {
		return Principal{}, &RateLimitError{retry}
	}
	// At most two Argon2 login verifications consume memory concurrently.
	select {
	case s.passwordSlots <- struct{}{}:
		defer func() { <-s.passwordSlots }()
	default:
		return Principal{}, &RateLimitError{time.Second}
	}
	var p Principal
	var hash string
	var disabled bool
	err := s.store.DB.QueryRow(`SELECT id, username, password_hash, role, disabled FROM users WHERE username = ? COLLATE NOCASE`, strings.TrimSpace(username)).Scan(&p.ID, &p.Username, &hash, &p.Role, &disabled)
	valid := false
	if err == nil {
		matches := verifyPassword(hash, password)
		valid = matches && !disabled
	}
	if !valid {
		if err != nil {
			_ = verifyPassword(dummyPasswordHash, password)
		}
		retry := s.limits.fail(keyIP, keyUser, s.now())
		log.Printf("security event=login_failed ip=%q username=%q", keyIP, strings.TrimSpace(username))
		if retry > 0 {
			return Principal{}, &RateLimitError{retry}
		}
		return Principal{}, ErrInvalidCredentials
	}
	token, err := randomToken()
	if err != nil {
		return Principal{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return Principal{}, err
	}
	sessionID, err := randomID()
	if err != nil {
		return Principal{}, err
	}
	now := s.now().UTC()
	csrfValue := base64.RawURLEncoding.EncodeToString(csrf)
	_, err = s.store.DB.Exec(`INSERT INTO sessions (id, session_hash, csrf_hash, user_id, created_at, last_seen_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, sessionID, digestBytes(token), digestBytes([]byte(csrfValue)), p.ID, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Add(absoluteTTL).Format(time.RFC3339Nano))
	if err != nil {
		return Principal{}, fmt.Errorf("Session speichern: %w", err)
	}
	s.limits.success(keyIP, keyUser)
	p.Session = sessionID
	p.csrfHash = digest([]byte(csrfValue))
	s.setCookies(w, token, csrf)
	log.Printf("security event=login_success user_id=%d username=%q ip=%q", p.ID, p.Username, keyIP)
	return p, nil
}

func (s *Service) Authenticate(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	token, err := cookieToken(r, SessionCookie)
	if err != nil {
		return Principal{}, false
	}
	now := s.now().UTC()
	var p Principal
	var csrfHash []byte
	var lastSeen, expires string
	err = s.store.DB.QueryRow(`SELECT u.id, u.username, u.role, s.id, s.csrf_hash, s.last_seen_at, s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.session_hash=? AND u.disabled=0`, digestBytes(token)).Scan(&p.ID, &p.Username, &p.Role, &p.Session, &csrfHash, &lastSeen, &expires)
	if err != nil {
		return Principal{}, false
	}
	if len(csrfHash) != sha256.Size {
		return Principal{}, false
	}
	copy(p.csrfHash[:], csrfHash)
	last, lastErr := time.Parse(time.RFC3339Nano, lastSeen)
	expiry, expiryErr := time.Parse(time.RFC3339Nano, expires)
	if lastErr != nil || expiryErr != nil || !now.Before(expiry) || now.Sub(last) >= idleTimeout {
		_, _ = s.store.DB.Exec(`DELETE FROM sessions WHERE id=?`, p.Session)
		clearCookies(w)
		log.Printf("security event=session_expired session_id=%q", p.Session)
		return Principal{}, false
	}
	if now.Sub(last) >= time.Minute {
		_, _ = s.store.DB.Exec(`UPDATE sessions SET last_seen_at=? WHERE id=?`, now.Format(time.RFC3339Nano), p.Session)
	}
	// The browser cookie has the same sliding lifetime as the server-side idle
	// timer. A missing or altered CSRF cookie is rotated without exposing the
	// session token to JavaScript.
	csrfValue := ""
	if cookie, err := r.Cookie(CSRFCookie); err == nil {
		cookieHash := digest([]byte(cookie.Value))
		if subtle.ConstantTimeCompare(cookieHash[:], p.csrfHash[:]) == 1 {
			csrfValue = cookie.Value
		}
	}
	if csrfValue == "" {
		csrf, err := randomToken()
		if err == nil {
			csrfValue = base64.RawURLEncoding.EncodeToString(csrf)
			p.csrfHash = digest([]byte(csrfValue))
			_, _ = s.store.DB.Exec(`UPDATE sessions SET csrf_hash=? WHERE id=?`, digestBytes([]byte(csrfValue)), p.Session)
		}
	}
	setCookie(w, SessionCookie, base64.RawURLEncoding.EncodeToString(token), true, int(idleTimeout.Seconds()))
	if csrfValue != "" {
		setCookie(w, CSRFCookie, csrfValue, false, int(idleTimeout.Seconds()))
	}
	return p, true
}

func (s *Service) ValidCSRF(p Principal, r *http.Request) bool {
	provided := r.Header.Get(CSRFHeader)
	if provided == "" {
		return false
	}
	got := digest([]byte(provided))
	return subtle.ConstantTimeCompare(got[:], p.csrfHash[:]) == 1
}
func (s *Service) Logout(w http.ResponseWriter, p Principal) error {
	if _, err := s.store.DB.Exec(`DELETE FROM sessions WHERE id=?`, p.Session); err != nil {
		return fmt.Errorf("Session beenden: %w", err)
	}
	clearCookies(w)
	log.Printf("security event=logout user_id=%d", p.ID)
	return nil
}

func (s *Service) ClearCookies(w http.ResponseWriter) { clearCookies(w) }
func (s *Service) ChangePassword(p Principal, current, next string) error {
	var hash string
	if err := s.store.DB.QueryRow(`SELECT password_hash FROM users WHERE id=? AND disabled=0`, p.ID).Scan(&hash); err != nil || !verifyPassword(hash, current) {
		return ErrInvalidCredentials
	}
	newHash, err := HashPassword(next)
	if err != nil {
		return err
	}
	tx, err := s.store.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`, newHash, s.now().UTC().Format(time.RFC3339Nano), p.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id=?`, p.ID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	log.Printf("security event=password_changed user_id=%d", p.ID)
	return nil
}
func (s *Service) DeleteExpiredSessions() error {
	now := s.now().UTC()
	result, err := s.store.DB.Exec(`DELETE FROM sessions WHERE expires_at <= ? OR last_seen_at <= ?`, now.Format(time.RFC3339Nano), now.Add(-idleTimeout).Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err == nil && count > 0 {
		log.Printf("security event=session_expired count=%d", count)
	}
	return nil
}

var ErrInvalidCredentials = errors.New("Benutzername oder Passwort ungültig.")

type RateLimitError struct{ RetryAfter time.Duration }

func (e *RateLimitError) Error() string {
	return "Zu viele Anmeldeversuche. Bitte später erneut versuchen."
}
func randomToken() ([]byte, error) { v := make([]byte, 32); _, e := rand.Read(v); return v, e }
func randomID() (string, error) {
	v := make([]byte, 16)
	_, e := rand.Read(v)
	return base64.RawURLEncoding.EncodeToString(v), e
}
func digest(v []byte) [32]byte    { return sha256.Sum256(v) }
func digestBytes(v []byte) []byte { sum := digest(v); return sum[:] }
func cookieToken(r *http.Request, name string) ([]byte, error) {
	c, e := r.Cookie(name)
	if e != nil {
		return nil, e
	}
	v, e := base64.RawURLEncoding.DecodeString(c.Value)
	if e != nil || len(v) != 32 {
		return nil, errors.New("invalid cookie")
	}
	return v, nil
}
func (s *Service) setCookies(w http.ResponseWriter, token, csrf []byte) {
	setCookie(w, SessionCookie, base64.RawURLEncoding.EncodeToString(token), true, int(idleTimeout.Seconds()))
	setCookie(w, CSRFCookie, base64.RawURLEncoding.EncodeToString(csrf), false, int(idleTimeout.Seconds()))
}
func setCookie(w http.ResponseWriter, name, value string, httpOnly bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: maxAge, HttpOnly: httpOnly, Secure: true, SameSite: http.SameSiteLaxMode})
}
func clearCookies(w http.ResponseWriter) {
	setCookie(w, SessionCookie, "", true, -1)
	setCookie(w, CSRFCookie, "", false, -1)
}

type limiter struct {
	mu          sync.Mutex
	entries     map[string]limitState
	lastCleanup time.Time
}
type limitState struct {
	failures            int
	start, delayedUntil time.Time
}

func newLimiter() *limiter { return &limiter{entries: map[string]limitState{}} }
func (l *limiter) retry(ip, u string, n time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(n)
	if len(l.entries) >= 10000 && (l.entries["ip:"+ip].start.IsZero() || l.entries["user:"+u].start.IsZero()) {
		return time.Minute
	}
	return maxDelay(l.entries["ip:"+ip], l.entries["user:"+u], n)
}
func (l *limiter) fail(ip, u string, n time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(n)
	needed := 0
	for _, key := range []string{"ip:" + ip, "user:" + u} {
		if l.entries[key].start.IsZero() {
			needed++
		}
	}
	if len(l.entries)+needed > 10000 {
		return time.Minute
	}
	return maxDelay(l.record("ip:"+ip, n), l.record("user:"+u, n), n)
}

func (l *limiter) cleanup(now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < time.Minute {
		return
	}
	for key, state := range l.entries {
		if now.Sub(state.start) >= 10*time.Minute {
			delete(l.entries, key)
		}
	}
	l.lastCleanup = now
}
func (l *limiter) record(k string, n time.Time) limitState {
	s := l.entries[k]
	if s.start.IsZero() || n.Sub(s.start) >= time.Minute {
		s = limitState{start: n}
	}
	s.failures++
	if s.failures >= 5 {
		delay := 1 << min(s.failures-5, 5)
		s.delayedUntil = n.Add(time.Duration(delay) * time.Second)
	}
	l.entries[k] = s
	return s
}
func (l *limiter) success(ip, u string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, "ip:"+ip)
	delete(l.entries, "user:"+u)
}
func maxDelay(a, b limitState, n time.Time) time.Duration {
	until := a.delayedUntil
	if b.delayedUntil.After(until) {
		until = b.delayedUntil
	}
	for _, state := range []limitState{a, b} {
		if state.failures >= 5 && n.Before(state.start.Add(time.Minute)) && state.start.Add(time.Minute).After(until) {
			until = state.start.Add(time.Minute)
		}
	}
	if n.Before(until) {
		return until.Sub(n)
	}
	return 0
}

const dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=2$MDEyMzQ1Njc4OWFiY2RlZg$BwTC5qjZ3UnTGh9V5z4y9g0/Fd80EDn3zXaKTvi3SkM"
