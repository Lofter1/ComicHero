package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func TestPublicAccessDefaultsAndAdminCanUpdate(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)`); err != nil {
		t.Fatalf("seed reader user: %v", err)
	}

	enabled, err := publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled default: %v", err)
	}
	if enabled {
		t.Fatal("publicAccessEnabled default = true; want false")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updatePublicAccess(readerCtx, db, UpdatePublicAccessPayload{Enabled: true}); err == nil {
		t.Fatal("non-admin updatePublicAccess returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: true})
	if err != nil {
		t.Fatalf("updatePublicAccess enable: %v", err)
	}
	if !output.Body.PublicAccess {
		t.Fatalf("output publicAccess = false; want true")
	}
	enabled, err = publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled after enable: %v", err)
	}
	if !enabled {
		t.Fatal("publicAccessEnabled after enable = false; want true")
	}

	if _, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: false}); err != nil {
		t.Fatalf("updatePublicAccess disable: %v", err)
	}
	enabled, err = publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled after disable: %v", err)
	}
	if enabled {
		t.Fatal("publicAccessEnabled after disable = true; want false")
	}
}

func TestPublicAccessAllowsAnonymousReadOnlyRoutes(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO reading_orders (id, name, author_user_id) VALUES (1, 'Public Order', 1);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position) VALUES (1, 1, 1);
	`); err != nil {
		t.Fatalf("seed public library data: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)
	RegisterComicRoutes(api, db, nil)
	RegisterReadingOrderRoutes(api, db, nil)

	disabled := httptest.NewRecorder()
	router.ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if disabled.Code != http.StatusUnauthorized {
		t.Fatalf("public disabled comics status = %d; want 401", disabled.Code)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: true}); err != nil {
		t.Fatalf("enable public access: %v", err)
	}

	comics := httptest.NewRecorder()
	router.ServeHTTP(comics, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if comics.Code != http.StatusOK {
		t.Fatalf("public comics status = %d; want 200: %s", comics.Code, comics.Body.String())
	}
	if !strings.Contains(comics.Body.String(), `"read":true`) {
		t.Fatalf("public comics body = %s; want default user's read status", comics.Body.String())
	}

	order := httptest.NewRecorder()
	router.ServeHTTP(order, httptest.NewRequest(http.MethodGet, "/api/readingOrders/1", nil))
	if order.Code != http.StatusOK {
		t.Fatalf("public reading order status = %d; want 200: %s", order.Code, order.Body.String())
	}
	if strings.Contains(order.Body.String(), `"canEdit":true`) {
		t.Fatalf("public reading order body = %s; want canEdit false", order.Body.String())
	}

	cbl := httptest.NewRecorder()
	router.ServeHTTP(cbl, httptest.NewRequest(http.MethodGet, "/api/readingOrders/1/cbl", nil))
	if cbl.Code != http.StatusOK {
		t.Fatalf("public CBL export status = %d; want 200: %s", cbl.Code, cbl.Body.String())
	}
	if !strings.Contains(cbl.Body.String(), "Public Order") {
		t.Fatalf("public CBL export body = %s; want reading order name", cbl.Body.String())
	}

	mutate := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/comic/1/read", strings.NewReader(`{"read":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(mutate, req)
	if mutate.Code != http.StatusUnauthorized {
		t.Fatalf("public mutation status = %d; want 401", mutate.Code)
	}
}

func TestExpiredSessionTokenIsRejected(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO user_sessions (token, user_id, expires_at)
		VALUES ('expired-session', 1, ?)
	`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed expired session: %v", err)
	}

	if userID, err := userIDFromSessionToken(context.Background(), db, "expired-session"); err == nil {
		t.Fatalf("expired session returned user %d, nil error; want error", userID)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM user_sessions WHERE token = 'expired-session'`); err != nil {
		t.Fatalf("count expired sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expired session count = %d; want 0", count)
	}
}

func TestSessionCookiesDefaultToLocalHTTPAndAreConfigurable(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	t.Setenv("COOKIE_SECURE", "")

	cookie, err := createSession(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false by default for local HTTP")
	}
	if expired := expiredSessionCookie(context.Background()); expired.Secure {
		t.Fatal("expired session cookie Secure = true; want false by default for local HTTP")
	}

	t.Setenv("COOKIE_SECURE", "true")
	cookie, err = createSession(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("createSession with COOKIE_SECURE=true: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("session cookie Secure = false; want true when COOKIE_SECURE=true")
	}
	if expired := expiredSessionCookie(context.Background()); !expired.Secure {
		t.Fatal("expired session cookie Secure = false; want true when COOKIE_SECURE=true")
	}
}

func TestSessionCookiesAutoUpgradeBehindHTTPSReverseProxy(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	t.Setenv("COOKIE_SECURE", "")

	plainCtx := context.WithValue(context.Background(), contextSecureRequestKey{}, false)
	cookie, err := createSession(plainCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false when the request arrived over plain HTTP")
	}

	httpsCtx := context.WithValue(context.Background(), contextSecureRequestKey{}, true)
	cookie, err = createSession(httpsCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession behind HTTPS proxy: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("session cookie Secure = false; want true when X-Forwarded-Proto/TLS indicated HTTPS")
	}

	// An explicit COOKIE_SECURE still wins over what the request looked like.
	t.Setenv("COOKIE_SECURE", "false")
	cookie, err = createSession(httpsCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession with COOKIE_SECURE=false: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false when COOKIE_SECURE=false overrides detection")
	}
}

func TestPasswordHashUsesCurrentIterationsAndVerifiesOldHashes(t *testing.T) {
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[1] != "600000" {
		t.Fatalf("hash = %q; want current iteration count encoded", hash)
	}
	if !checkPassword("secret1", hash) {
		t.Fatal("checkPassword rejected current hash")
	}

	salt := []byte("0123456789abcdef")
	key := derivePasswordKey([]byte("secret1"), salt, 120000, 32)
	oldHash := "pbkdf2_sha256$120000$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key)
	if !checkPassword("secret1", oldHash) {
		t.Fatal("checkPassword rejected old iteration count")
	}
	if !passwordHashNeedsUpgrade(oldHash) {
		t.Fatal("passwordHashNeedsUpgrade returned false for old hash")
	}
}
