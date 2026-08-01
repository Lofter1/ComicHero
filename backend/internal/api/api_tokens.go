package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

// apiTokenPrefix is prepended to every generated token so tokens are easy to
// recognize (in logs, in a .env file, etc.) and can plausibly be told apart
// from a session cookie value at a glance.
const apiTokenPrefix = "ch_pat_"

// apiTokenPrefixDisplayLen is how many characters of the raw token (prefix
// included) are stored and shown back to the user, so a token can be told
// apart from the user's other tokens without ever storing or displaying the
// full secret again after creation.
const apiTokenPrefixDisplayLen = len(apiTokenPrefix) + 6

type contextAPITokenAuthenticatedKey struct{}
type contextAPITokenScopesKey struct{}

// generateAPIToken creates a new raw token value. 32 random bytes
// URL-safe-base64-encoded gives 256 bits of entropy, comparable to the
// session tokens created in createSession.
func generateAPIToken() (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	return apiTokenPrefix + token, nil
}

func hashAPIToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeAPITokenScopes(scopes []string) ([]string, error) {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" || seen[scope] {
			continue
		}
		if !validAPITokenScopes[scope] {
			return nil, huma.Error422UnprocessableEntity("unknown scope: " + scope)
		}
		seen[scope] = true
		normalized = append(normalized, scope)
	}
	if len(normalized) == 0 {
		return nil, huma.Error422UnprocessableEntity("at least one scope is required")
	}
	sort.Strings(normalized)
	return normalized, nil
}

// apiTokenRouteScope reports the scope required to access a given
// method+path, and whether an API token can reach it at all. This is
// deliberately a narrow allowlist, not a blocklist: an API token is meant to
// hand an external service (a reading server, collection manager, etc.) a
// small, purpose-built slice of access - search reading orders, start one,
// mark a comic read, and fetch the next unread comic - not the same surface
// a logged-in browser session gets. Anything not listed here (account
// management, token management itself, deleting content, and so on) is
// unreachable with a token no matter what scopes it holds.
func apiTokenRouteScope(method, path string) (scope string, allowed bool) {
	path = strings.Trim(strings.TrimPrefix(path, "/api"), "/")
	parts := strings.Split(path, "/")

	switch {
	case method == http.MethodGet && len(parts) == 1 && parts[0] == "readingOrders":
		// Search/list reading orders (?q=... for text search).
		return apiTokenScopeReadingRead, true
	case method == http.MethodGet && len(parts) == 2 && parts[0] == "readingOrders" && positivePathID(parts[1]):
		// Reading order detail, including computed progress.
		return apiTokenScopeReadingRead, true
	case method == http.MethodGet && len(parts) == 3 && parts[0] == "readingOrders" && positivePathID(parts[1]) && parts[2] == "next":
		// Next unread comic in a reading order.
		return apiTokenScopeReadingRead, true
	case method == http.MethodPost && len(parts) == 3 && parts[0] == "readingOrders" && positivePathID(parts[1]) && parts[2] == "start":
		return apiTokenScopeReadingWrite, true
	case method == http.MethodDelete && len(parts) == 3 && parts[0] == "readingOrders" && positivePathID(parts[1]) && parts[2] == "start":
		// Stop reading, the counterpart of start above.
		return apiTokenScopeReadingWrite, true
	case method == http.MethodPatch && len(parts) == 3 && parts[0] == "comic" && positivePathID(parts[1]) && parts[2] == "read":
		// Mark a comic read/unread/skipped. Path is singular "comic" to
		// match the existing updateComicReadStatus route.
		return apiTokenScopeReadingWrite, true
	default:
		return "", false
	}
}

// APITokenMiddleware authenticates requests carrying an `Authorization:
// Bearer <token>` header against stored API tokens, independently of the
// browser session/cookie flow. It must run before UserMiddleware, which
// checks contextAPITokenAuthenticatedKey and skips its own session/user-mode
// gate once this middleware has already established a user identity.
func APITokenMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(header, bearerPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			userID, scopes, err := authenticateAPIToken(r.Context(), db, token)
			if err != nil {
				http.Error(w, "invalid or expired API token", http.StatusUnauthorized)
				return
			}

			requiredScope, allowed := apiTokenRouteScope(r.Method, r.URL.Path)
			if !allowed {
				http.Error(w, "this API token cannot access this endpoint", http.StatusForbidden)
				return
			}
			if !scopes[requiredScope] {
				http.Error(w, "this API token does not have the "+requiredScope+" scope", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), contextAPITokenAuthenticatedKey{}, true)
			ctx = context.WithValue(ctx, contextUserIDKey{}, userID)
			ctx = context.WithValue(ctx, contextAPITokenScopesKey{}, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticateAPIToken(ctx context.Context, db *sqlx.DB, token string) (int, map[string]bool, error) {
	hash := hashAPIToken(token)

	var row struct {
		ID        int    `db:"id"`
		UserID    int    `db:"user_id"`
		Scopes    string `db:"scopes"`
		ExpiresAt string `db:"expires_at"`
		RevokedAt string `db:"revoked_at"`
	}
	// Looking up by the hash directly (rather than scanning every active
	// token and comparing) is safe here because the hash itself is a
	// high-entropy, one-way digest of the secret - it isn't the kind of
	// value a timing side-channel over which row matched could meaningfully
	// help an attacker guess.
	if err := db.GetContext(ctx, &row, `
		SELECT id, user_id, scopes, expires_at, revoked_at
		FROM api_tokens
		WHERE token_hash = ?
	`, hash); err != nil {
		return 0, nil, huma.Error401Unauthorized("invalid or expired API token")
	}
	if row.RevokedAt != "" {
		return 0, nil, huma.Error401Unauthorized("this API token has been revoked")
	}
	if row.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, row.ExpiresAt)
		if err == nil && !expiresAt.After(time.Now().UTC()) {
			return 0, nil, huma.Error401Unauthorized("this API token has expired")
		}
	}

	scopes := map[string]bool{}
	for _, scope := range strings.Split(row.Scopes, ",") {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			scopes[scope] = true
		}
	}

	_, _ = db.ExecContext(ctx, `
		UPDATE api_tokens SET last_used_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339), row.ID)

	return row.UserID, scopes, nil
}

func listAPITokens(ctx context.Context, db *sqlx.DB) (*APITokenListOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	var rows []apiTokenRow
	if err := db.SelectContext(ctx, &rows, `
		SELECT id, name, token_prefix, scopes, created_at, last_used_at, expires_at, revoked_at
		FROM api_tokens
		WHERE user_id = ? AND revoked_at = ''
		ORDER BY created_at DESC
	`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch API tokens")
	}
	tokens := make([]APIToken, 0, len(rows))
	for _, row := range rows {
		token := row.APIToken
		token.Scopes = splitAPITokenScopes(row.ScopesRaw)
		tokens = append(tokens, token)
	}
	return &APITokenListOutput{Body: tokens}, nil
}

func createAPIToken(ctx context.Context, db *sqlx.DB, payload CreateAPITokenPayload) (*CreateAPITokenOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return nil, huma.Error422UnprocessableEntity("name is required")
	}
	scopes, err := normalizeAPITokenScopes(payload.Scopes)
	if err != nil {
		return nil, err
	}
	expiresAt := ""
	if strings.TrimSpace(payload.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("expiresAt must be an RFC3339 timestamp")
		}
		if !parsed.After(time.Now().UTC()) {
			return nil, huma.Error422UnprocessableEntity("expiresAt must be in the future")
		}
		expiresAt = parsed.UTC().Format(time.RFC3339)
	}

	rawToken, err := generateAPIToken()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create API token")
	}
	prefix := rawToken
	if len(prefix) > apiTokenPrefixDisplayLen {
		prefix = prefix[:apiTokenPrefixDisplayLen]
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO api_tokens (user_id, name, token_prefix, token_hash, scopes, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, name, prefix, hashAPIToken(rawToken), strings.Join(scopes, ","), expiresAt)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create API token")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create API token")
	}

	var createdAt string
	_ = db.GetContext(ctx, &createdAt, `SELECT created_at FROM api_tokens WHERE id = ?`, id)

	return &CreateAPITokenOutput{Body: APITokenCreated{
		APIToken: APIToken{
			ID:          int(id),
			Name:        name,
			TokenPrefix: prefix,
			Scopes:      scopes,
			CreatedAt:   createdAt,
			ExpiresAt:   expiresAt,
		},
		Token: rawToken,
	}}, nil
}

func revokeAPIToken(ctx context.Context, db *sqlx.DB, id int) (*struct{}, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, `
		UPDATE api_tokens SET revoked_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at = ''
	`, time.Now().UTC().Format(time.RFC3339), id, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to revoke API token")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to revoke API token")
	}
	if affected == 0 {
		return nil, huma.Error404NotFound("API token not found")
	}
	return nil, nil
}

func splitAPITokenScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes
}

func RegisterAPITokenRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "listAPITokens",
		Tags:        []string{tagAPITokens},
		Summary:     "List API tokens",
		Description: "Returns the current user's active (non-revoked) API tokens. Token values themselves are never returned after creation - only each token's prefix, scopes, and usage metadata.",
		Method:      http.MethodGet,
		Path:        "/tokens",
		Errors:      errsRead,
	}, func(ctx context.Context, _ *struct{}) (*APITokenListOutput, error) {
		return listAPITokens(ctx, db)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "createAPIToken",
		Tags:          []string{tagAPITokens},
		Summary:       "Create an API token",
		Description:   "Creates a new scoped API token for the current user, for connecting an external service such as a reading server or collection manager. The raw token is returned once, in this response only, and cannot be retrieved again.",
		Method:        http.MethodPost,
		Path:          "/tokens",
		DefaultStatus: 201,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *CreateAPITokenInput) (*CreateAPITokenOutput, error) {
		return createAPIToken(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "revokeAPIToken",
		Tags:          []string{tagAPITokens},
		Summary:       "Revoke an API token",
		Description:   "Immediately revokes an API token belonging to the current user. Revoked tokens can no longer authenticate requests.",
		Method:        http.MethodDelete,
		Path:          "/tokens/{id}",
		DefaultStatus: 204,
		Errors:        errsWrite,
	}, func(ctx context.Context, input *RevokeAPITokenInput) (*struct{}, error) {
		return revokeAPIToken(ctx, db, input.ID)
	})
}
