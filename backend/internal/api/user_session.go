package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type loginRateLimitEntry struct {
	count      int
	windowEnds time.Time
}

type loginRateLimiter struct {
	maxAttempts int
	window      time.Duration
	mu          sync.Mutex
	attempts    map[string]loginRateLimitEntry
}

func (l *loginRateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := l.attempts[key]
	if entry.windowEnds.IsZero() || !now.Before(entry.windowEnds) {
		l.attempts[key] = loginRateLimitEntry{count: 1, windowEnds: now.Add(l.window)}
		return true
	}
	if entry.count >= l.maxAttempts {
		return false
	}
	entry.count++
	l.attempts[key] = entry
	return true
}

var (
	authLoginLimiter        = newLoginRateLimiter(loginRateLimitMaxAttempts, loginRateLimitWindow)
	authRegistrationLimiter = newLoginRateLimiter(registrationRateLimitMaxAttempts, registrationRateLimitWindow)
)

func clientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		if ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func cookieHeader(cookie *http.Cookie) []http.Cookie {
	if cookie == nil {
		return nil
	}
	return []http.Cookie{*cookie}
}

func createSession(ctx context.Context, db *sqlx.DB, userID int) (*http.Cookie, error) {
	token, err := randomToken(32)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create session")
	}
	expiresAt := time.Now().UTC().Add(sessionTTL)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, expiresAt.Format(time.RFC3339)); err != nil {
		return nil, huma.Error500InternalServerError("failed to save session")
	}
	return &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt, HttpOnly: true, Secure: secureSessionCookies(), SameSite: http.SameSiteLaxMode}, nil
}

func currentUserID(ctx context.Context) (int, error) {
	userID, ok := ctx.Value(contextUserIDKey{}).(int)
	if !ok || userID <= 0 {
		return 0, huma.Error401Unauthorized("login required")
	}
	return userID, nil
}

func currentUserIsPublic(ctx context.Context) bool {
	public, _ := ctx.Value(contextPublicAccessKey{}).(bool)
	return public
}

func expiredSessionCookie() *http.Cookie {
	return &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureSessionCookies(), SameSite: http.SameSiteLaxMode}
}

func isLoginRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/login"
}

func isPublicReadRequest(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		switch parts[0] {
		case "readingOrders", "comics", "series", "characters", "arcs":
			return true
		default:
			return false
		}
	}
	if len(parts) == 2 && positivePathID(parts[1]) {
		switch parts[0] {
		case "readingOrders", "comics", "series", "characters", "arcs":
			return true
		default:
			return false
		}
	}
	return len(parts) == 3 && parts[0] == "readingOrders" && parts[2] == "cbl" && positivePathID(parts[1])
}

func isRegisterRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/register"
}

func isUserRouteAllowedWithoutSession(path string) bool {
	path = strings.TrimPrefix(path, "/api")
	switch path {
	case "/auth/status", "/auth/setup", "/auth/register", "/auth/login", "/openapi.json", "/openapi.yaml", "/docs":
		return true
	default:
		return false
	}
}

func newLoginRateLimiter(maxAttempts int, window time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		attempts:    map[string]loginRateLimitEntry{},
	}
}

func secureSessionCookies() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func sessionUserID(r *http.Request, db *sqlx.DB) (int, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return 0, huma.Error401Unauthorized("login required")
	}
	userID, err := userIDFromSessionToken(r.Context(), db, cookie.Value)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func userIDFromSessionToken(ctx context.Context, db *sqlx.DB, token string) (int, error) {
	if token == "" {
		return 0, huma.Error401Unauthorized("login required")
	}
	var row struct {
		UserID    int    `db:"user_id"`
		ExpiresAt string `db:"expires_at"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT user_id, expires_at FROM user_sessions WHERE token = ?
	`, token); err != nil {
		return 0, huma.Error401Unauthorized("login required")
	}
	expiresAt, err := time.Parse(time.RFC3339, row.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now().UTC()) {
		_, _ = db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
		return 0, huma.Error401Unauthorized("login required")
	}
	return row.UserID, nil
}

func randomBytes(size int) ([]byte, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}

func randomToken(size int) (string, error) {
	value, err := randomBytes(size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
