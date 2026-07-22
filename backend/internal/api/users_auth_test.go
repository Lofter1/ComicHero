package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func TestMultiUserSetupSetsSessionCookieForProtectedRoutes(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)
	RegisterComicRoutes(api, db, nil)

	setup := httptest.NewRecorder()
	setupBody := strings.NewReader(`{"mode":"multi","name":"Test","email":"test@example.com","password":"secret1"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(setup, setupReq)
	if setup.Code != http.StatusOK {
		t.Fatalf("setup status = %d; want 200: %s", setup.Code, setup.Body.String())
	}
	cookies := setup.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value == "" {
		t.Fatalf("setup cookies = %#v; want %s session cookie", cookies, sessionCookieName)
	}

	statusRecorder := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusReq.AddCookie(cookies[0])
	router.ServeHTTP(statusRecorder, statusReq)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status with cookie status = %d; want 200: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var status UserStatus
	if err := json.NewDecoder(statusRecorder.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.User == nil || !status.User.IsAdmin {
		t.Fatalf("status user = %#v; want admin user", status.User)
	}
	if !status.MetronPermissions.Allowed || !metronScopeAllowed(status.MetronPermissions.Scopes, metronScopeMonitor) {
		t.Fatalf("status metron permissions = %#v; want monitor access for admin", status.MetronPermissions)
	}

	withoutCookie := httptest.NewRecorder()
	router.ServeHTTP(withoutCookie, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if withoutCookie.Code != http.StatusUnauthorized {
		t.Fatalf("comics without cookie status = %d; want 401", withoutCookie.Code)
	}

	withCookie := httptest.NewRecorder()
	comicsReq := httptest.NewRequest(http.MethodGet, "/api/comics", nil)
	comicsReq.AddCookie(cookies[0])
	router.ServeHTTP(withCookie, comicsReq)
	if withCookie.Code != http.StatusOK {
		t.Fatalf("comics with cookie status = %d; want 200: %s", withCookie.Code, withCookie.Body.String())
	}
	if !strings.Contains(withCookie.Body.String(), `"series":"Amazing Spider-Man"`) {
		t.Fatalf("comics body = %s; want seeded comic", withCookie.Body.String())
	}
}

func TestLoginRateLimitReturnsTooManyRequests(t *testing.T) {
	previousLimiter := authLoginLimiter
	authLoginLimiter = newLoginRateLimiter(loginRateLimitMaxAttempts, loginRateLimitWindow)
	t.Cleanup(func() {
		authLoginLimiter = previousLimiter
	})

	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed login user: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)

	for i := 0; i < loginRateLimitMaxAttempts; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.44:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = 429; want not rate-limited yet", i+1)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.44:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited login status = %d; want 429: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginUsesEmailAddress(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed login user: %v", err)
	}

	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Name:     "Test",
		Password: "secret1",
	}); err == nil {
		t.Fatal("loginUser accepted a username without an email")
	}

	output, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Email:    "Test@Example.com",
		Password: "secret1",
	})
	if err != nil {
		t.Fatalf("loginUser with email: %v", err)
	}
	if output.Body.User == nil || output.Body.User.Email != "test@example.com" {
		t.Fatalf("user = %#v; want logged-in user by normalized email", output.Body.User)
	}
}

func TestResendEmailVerificationAuthenticatesBeforeRotatingToken(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users
		SET name = 'Pending', email = 'pending@example.com', email_verified_at = '', password_hash = ?
		WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed pending user: %v", err)
	}

	oldToken, _, err := createEmailVerification(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("createEmailVerification: %v", err)
	}
	oldHash := emailVerificationTokenHash(oldToken)

	if _, err := resendEmailVerification(context.Background(), db, ResendEmailVerificationPayload{
		Email: "pending@example.com", Password: "wrong",
	}); err == nil {
		t.Fatal("resendEmailVerification accepted an incorrect password")
	}
	var usedAt string
	if err := db.Get(&usedAt, `SELECT used_at FROM user_email_verifications WHERE token_hash = ?`, oldHash); err != nil {
		t.Fatalf("fetch original verification token: %v", err)
	}
	if usedAt != "" {
		t.Fatalf("failed authentication consumed the original token at %q", usedAt)
	}

	output, err := resendEmailVerification(context.Background(), db, ResendEmailVerificationPayload{
		Email: "Pending@Example.com", Password: "secret1",
	})
	if err != nil {
		t.Fatalf("resendEmailVerification: %v", err)
	}
	if !output.Body.EmailVerificationRequired || output.Body.EmailVerificationEmail != "pending@example.com" {
		t.Fatalf("status = %#v; want pending email verification", output.Body)
	}
	if err := db.Get(&usedAt, `SELECT used_at FROM user_email_verifications WHERE token_hash = ?`, oldHash); err != nil {
		t.Fatalf("fetch rotated verification token: %v", err)
	}
	if usedAt == "" {
		t.Fatal("successful resend did not consume the original token")
	}
	var activeTokens int
	if err := db.Get(&activeTokens, `SELECT COUNT(*) FROM user_email_verifications WHERE user_id = 1 AND used_at = ''`); err != nil {
		t.Fatalf("count active verification tokens: %v", err)
	}
	if activeTokens != 1 {
		t.Fatalf("active verification tokens = %d; want 1", activeTokens)
	}
}

func TestExpiredAccountTokensAreRejected(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	verificationToken := "expired-verification"
	resetToken := "expired-reset"
	if _, err := db.Exec(`
		INSERT INTO user_email_verifications (token_hash, user_id, expires_at)
		VALUES (?, 1, '2000-01-01T00:00:00Z');
		INSERT INTO user_password_resets (token_hash, user_id, expires_at)
		VALUES (?, 1, '2000-01-01T00:00:00Z');
	`, emailVerificationTokenHash(verificationToken), emailVerificationTokenHash(resetToken)); err != nil {
		t.Fatalf("seed expired tokens: %v", err)
	}

	if _, err := verifyEmailToken(context.Background(), db, verificationToken); err == nil {
		t.Fatal("verifyEmailToken accepted an expired token")
	}
	if _, err := verifyPasswordResetToken(context.Background(), db, resetToken); err == nil {
		t.Fatal("verifyPasswordResetToken accepted an expired token")
	}
}

func TestPasswordResetRequestDoesNotRevealUnknownEmail(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}

	output, err := requestPasswordReset(context.Background(), db, ForgotPasswordPayload{
		Email: "missing@example.com",
	})
	if err != nil {
		t.Fatalf("requestPasswordReset for unknown email: %v", err)
	}
	if output.Body.Mode != userModeMulti || output.Body.User != nil {
		t.Fatalf("status = %#v; want anonymous multi-user status", output.Body)
	}
	var tokenCount int
	if err := db.Get(&tokenCount, `SELECT COUNT(*) FROM user_password_resets`); err != nil {
		t.Fatalf("count password reset tokens: %v", err)
	}
	if tokenCount != 0 {
		t.Fatalf("password reset token count = %d; want 0", tokenCount)
	}
}

func TestLogoutDeletesOnlyCurrentSessionAndExpiresCookie(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO user_sessions (token, user_id, expires_at) VALUES
			('current-session', 1, '2999-01-01T00:00:00Z'),
			('other-session', 1, '2999-01-01T00:00:00Z');
	`); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	output, err := logoutUser(context.Background(), db, "current-session")
	if err != nil {
		t.Fatalf("logoutUser: %v", err)
	}
	if len(output.SetCookie) != 1 || output.SetCookie[0].MaxAge >= 0 {
		t.Fatalf("cookies = %#v; want expired session cookie", output.SetCookie)
	}
	var currentCount, otherCount int
	if err := db.Get(&currentCount, `SELECT COUNT(*) FROM user_sessions WHERE token = 'current-session'`); err != nil {
		t.Fatalf("count current session: %v", err)
	}
	if err := db.Get(&otherCount, `SELECT COUNT(*) FROM user_sessions WHERE token = 'other-session'`); err != nil {
		t.Fatalf("count other session: %v", err)
	}
	if currentCount != 0 || otherCount != 1 {
		t.Fatalf("session counts = current %d, other %d; want 0 and 1", currentCount, otherCount)
	}
}

func TestRegistrationRateLimitReturnsTooManyRequests(t *testing.T) {
	previousLimiter := authRegistrationLimiter
	authRegistrationLimiter = newLoginRateLimiter(registrationRateLimitMaxAttempts, registrationRateLimitWindow)
	t.Cleanup(func() {
		authRegistrationLimiter = previousLimiter
	})

	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO app_settings (key, value) VALUES ('registration_mode', 'open');
	`); err != nil {
		t.Fatalf("seed open registration mode: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)

	for i := 0; i < registrationRateLimitMaxAttempts; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.45:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = 429; want not rate-limited yet", i+1)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.45:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited registration status = %d; want 429: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequireAdminUserFailsWithoutUserContext(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	if userID, err := requireAdminUser(context.Background(), db); err == nil {
		t.Fatalf("requireAdminUser without user context = %d, nil; want error", userID)
	}
}

func TestPerUserEndpointFailsWithoutUserContext(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterComicRoutes(api, db, nil)

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"read":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/comic/1/read", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("read-status without user context status = %d; want 401: %s", recorder.Code, recorder.Body.String())
	}
}
