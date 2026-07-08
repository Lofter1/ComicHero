package api

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const (
	metronScopeAll     = "*"
	metronScopeSearch  = "search"
	metronScopeDetail  = "detail"
	metronScopeImport  = "import"
	metronScopeMonitor = "monitor"
)

var validMetronScopes = map[string]bool{
	metronScopeAll:     true,
	metronScopeSearch:  true,
	metronScopeDetail:  true,
	metronScopeImport:  true,
	metronScopeMonitor: true,
}

func listUsers(ctx context.Context, db *sqlx.DB) (*UserListOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}

	var rows []struct {
		ID          int            `db:"id"`
		Name        string         `db:"name"`
		IsAdmin     bool           `db:"is_admin"`
		Allowed     sql.NullBool   `db:"allowed"`
		Scopes      sql.NullString `db:"scopes"`
		HourlyLimit sql.NullInt64  `db:"hourly_limit"`
	}
	if err := db.SelectContext(ctx, &rows, `
		SELECT
			u.id,
			u.name,
			u.is_admin,
			p.allowed,
			p.scopes,
			p.hourly_limit
		FROM users u
		LEFT JOIN user_metron_permissions p ON p.user_id = u.id
		ORDER BY u.name
	`); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch users")
	}

	users := make([]UserAdminView, 0, len(rows))
	for _, row := range rows {
		users = append(users, UserAdminView{
			User: User{
				ID:      row.ID,
				Name:    row.Name,
				IsAdmin: row.IsAdmin,
			},
			MetronPermissions: permissionsFromRow(row.IsAdmin, row.Allowed, row.Scopes, row.HourlyLimit),
		})
	}
	return &UserListOutput{Body: users}, nil
}

func updateUserMetronPermissions(ctx context.Context, db *sqlx.DB, userID int, payload UserMetronPermissions) (*UserAdminOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, huma.Error400BadRequest("user id is required")
	}

	user, err := getUserByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	permissions, err := normalizeMetronPermissions(payload)
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			allowed = excluded.allowed,
			scopes = excluded.scopes,
			hourly_limit = excluded.hourly_limit,
			updated_at = CURRENT_TIMESTAMP
	`, userID, permissions.Allowed, strings.Join(permissions.Scopes, ","), permissions.HourlyLimit); err != nil {
		return nil, huma.Error500InternalServerError("failed to save Metron permissions")
	}

	return &UserAdminOutput{Body: UserAdminView{User: user, MetronPermissions: permissions}}, nil
}

func metronPermissionsForUser(ctx context.Context, db *sqlx.DB, userID int) (UserMetronPermissions, error) {
	var row struct {
		IsAdmin     bool           `db:"is_admin"`
		Allowed     sql.NullBool   `db:"allowed"`
		Scopes      sql.NullString `db:"scopes"`
		HourlyLimit sql.NullInt64  `db:"hourly_limit"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT
			u.is_admin,
			p.allowed,
			p.scopes,
			p.hourly_limit
		FROM users u
		LEFT JOIN user_metron_permissions p ON p.user_id = u.id
		WHERE u.id = ?
	`, userID); err != nil {
		if err == sql.ErrNoRows {
			return UserMetronPermissions{}, huma.Error401Unauthorized("login required")
		}
		return UserMetronPermissions{}, huma.Error500InternalServerError("failed to fetch Metron permissions")
	}
	return permissionsFromRow(row.IsAdmin, row.Allowed, row.Scopes, row.HourlyLimit), nil
}

func authorizeMetron(ctx context.Context, db *sqlx.DB, scope, endpoint string) error {
	if db == nil {
		return nil
	}
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}

	permissions, err := metronPermissionsForUser(ctx, db, userID)
	if err != nil {
		return err
	}

	if !permissions.Allowed {
		return huma.Error403Forbidden("Metron access is not allowed for this user")
	}
	if !metronScopeAllowed(permissions.Scopes, scope) {
		return huma.Error403Forbidden("Metron endpoint is not allowed for this user")
	}
	if permissions.HourlyLimit > 0 {
		var recent int
		if err := db.GetContext(ctx, &recent, `
			SELECT COUNT(*)
			FROM user_metron_request_log
			WHERE user_id = ?
			  AND created_at >= datetime('now', '-1 hour')
		`, userID); err != nil {
			return huma.Error500InternalServerError("failed to check Metron usage limit")
		}
		if recent >= permissions.HourlyLimit {
			return huma.Error429TooManyRequests("Metron endpoint limit reached for this user")
		}
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_metron_request_log (user_id, scope, endpoint)
		VALUES (?, ?, ?)
	`, userID, scope, endpoint); err != nil {
		return huma.Error500InternalServerError("failed to record Metron usage")
	}
	return nil
}

func normalizeMetronPermissions(payload UserMetronPermissions) (UserMetronPermissions, error) {
	if payload.HourlyLimit < 0 {
		return UserMetronPermissions{}, huma.Error400BadRequest("hourlyLimit must be zero or greater")
	}
	scopes := normalizeMetronScopes(payload.Scopes)
	if payload.Allowed && len(scopes) == 0 {
		scopes = []string{metronScopeAll}
	}
	for _, scope := range scopes {
		if !validMetronScopes[scope] {
			return UserMetronPermissions{}, huma.Error400BadRequest("invalid Metron permission scope")
		}
	}
	return UserMetronPermissions{
		Allowed:     payload.Allowed,
		Scopes:      scopes,
		HourlyLimit: payload.HourlyLimit,
	}, nil
}

func permissionsFromRow(isAdmin bool, allowed sql.NullBool, scopes sql.NullString, hourlyLimit sql.NullInt64) UserMetronPermissions {
	if !allowed.Valid {
		if isAdmin {
			return UserMetronPermissions{Allowed: true, Scopes: []string{metronScopeAll}}
		}
		return UserMetronPermissions{}
	}
	limit := 0
	if hourlyLimit.Valid {
		limit = int(hourlyLimit.Int64)
	}
	return UserMetronPermissions{
		Allowed:     allowed.Bool,
		Scopes:      normalizeMetronScopes(strings.Split(scopes.String, ",")),
		HourlyLimit: limit,
	}
}

func normalizeMetronScopes(values []string) []string {
	seen := map[string]bool{}
	scopes := []string{}
	for _, value := range values {
		scope := strings.TrimSpace(value)
		if scope == "" || seen[scope] {
			continue
		}
		if scope == metronScopeAll {
			return []string{metronScopeAll}
		}
		seen[scope] = true
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

func metronScopeAllowed(scopes []string, scope string) bool {
	for _, allowed := range scopes {
		if allowed == metronScopeAll || allowed == scope {
			return true
		}
	}
	return false
}
