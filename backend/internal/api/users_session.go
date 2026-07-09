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
	authLoginLimiter              = newLoginRateLimiter(loginRateLimitMaxAttempts, loginRateLimitWindow)
	authRegistrationLimiter       = newLoginRateLimiter(registrationRateLimitMaxAttempts, registrationRateLimitWindow)
	authEmailVerificationLimiter  = newLoginRateLimiter(emailVerificationRateLimitMaxAttempts, emailVerificationRateLimitWindow)
	authEmailVerificationResender = newLoginRateLimiter(emailVerificationResendMaxAttempts, emailVerificationResendWindow)
	authPasswordResetRequester    = newLoginRateLimiter(passwordResetRequestMaxAttempts, passwordResetRequestWindow)
	authPasswordResetLimiter      = newLoginRateLimiter(passwordResetMaxAttempts, passwordResetWindow)
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

type contextSecureRequestKey struct{}

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
	return &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expiresAt, HttpOnly: true, Secure: secureSessionCookies(ctx), SameSite: http.SameSiteLaxMode}, nil
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

func expiredSessionCookie(ctx context.Context) *http.Cookie {
	return &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: secureSessionCookies(ctx), SameSite: http.SameSiteLaxMode}
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

func isVerifyEmailRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/verify-email"
}

func isResendEmailVerificationRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/verify-email/resend"
}

func isPasswordResetRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/forgot-password"
}

func isResetPasswordRequest(r *http.Request) bool {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	return r.Method == http.MethodPost && path == "/auth/reset-password"
}

func isUserRouteAllowedWithoutSession(path string) bool {
	path = strings.TrimPrefix(path, "/api")
	switch path {
	case "/auth/status", "/auth/setup", "/auth/register", "/auth/login", "/auth/verify-email", "/auth/verify-email/resend", "/auth/forgot-password", "/auth/reset-password", "/openapi.json", "/openapi.yaml", "/docs":
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

// secureSessionCookies decides whether session cookies get the Secure flag.
// COOKIE_SECURE, when explicitly set, always wins so operators can force
// either behavior. Otherwise, when UserMiddleware detected that this request
// arrived over HTTPS (direct TLS, or a reverse proxy that set
// X-Forwarded-Proto: https), cookies are upgraded to Secure automatically -
// this matters because it's very easy to serve a public instance over HTTPS
// and forget to also set COOKIE_SECURE=true, silently leaving session
// cookies without the Secure flag. With no request context to inspect (e.g.
// a direct unit test), this keeps the historical default of false, matching
// a plain local HTTP setup.
func secureSessionCookies(ctx context.Context) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		secure, _ := ctx.Value(contextSecureRequestKey{}).(bool)
		return secure
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
		UserID          int    `db:"user_id"`
		ExpiresAt       string `db:"expires_at"`
		EmailVerifiedAt string `db:"email_verified_at"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT s.user_id, s.expires_at, u.email_verified_at
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?
	`, token); err != nil {
		return 0, huma.Error401Unauthorized("login required")
	}
	expiresAt, err := time.Parse(time.RFC3339, row.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now().UTC()) {
		_, _ = db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
		return 0, huma.Error401Unauthorized("login required")
	}
	if row.EmailVerifiedAt == "" {
		_, _ = db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token)
		return 0, huma.Error401Unauthorized("email verification required")
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

func UserMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secure := r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
			r = r.WithContext(context.WithValue(r.Context(), contextSecureRequestKey{}, secure))

			if isLoginRequest(r) && !authLoginLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many login attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isRegisterRequest(r) && !authRegistrationLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many registration attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isVerifyEmailRequest(r) && !authEmailVerificationLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many email verification attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isResendEmailVerificationRequest(r) && !authEmailVerificationResender.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many email verification emails requested, try again later", http.StatusTooManyRequests)
				return
			}
			if isPasswordResetRequest(r) && !authPasswordResetRequester.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many password reset emails requested, try again later", http.StatusTooManyRequests)
				return
			}
			if isResetPasswordRequest(r) && !authPasswordResetLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many password reset attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isUserRouteAllowedWithoutSession(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			mode, configured, err := userMode(r.Context(), db)
			if err != nil {
				http.Error(w, "failed to read user setup", http.StatusInternalServerError)
				return
			}
			if !configured {
				http.Error(w, "user setup required", http.StatusPreconditionRequired)
				return
			}

			var userID int
			publicAccess := false
			if mode == userModeSingle {
				userID, err = ensureDefaultUser(r.Context(), db)
			} else {
				userID, err = sessionUserID(r, db)
				if err != nil && isPublicReadRequest(r) {
					enabled, settingErr := publicAccessEnabled(r.Context(), db)
					if settingErr != nil {
						http.Error(w, "failed to read public access setting", http.StatusInternalServerError)
						return
					}
					if enabled {
						userID, err = ensureDefaultUser(r.Context(), db)
						publicAccess = err == nil
					}
				}
			}
			if err != nil {
				http.Error(w, "login required", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextUserIDKey{}, userID)
			if publicAccess {
				ctx = context.WithValue(ctx, contextPublicAccessKey{}, true)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
