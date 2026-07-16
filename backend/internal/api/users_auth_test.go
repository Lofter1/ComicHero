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
