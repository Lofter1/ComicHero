package api

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMetronPermissionsControlScopesAndHourlyLimit(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/comics"); err == nil {
		t.Fatal("authorizeMetron returned nil for reader without Metron permissions")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateUserMetronPermissions(adminCtx, db, 2, UserMetronPermissions{
		Allowed:     true,
		Scopes:      []string{metronScopeSearch},
		HourlyLimit: 1,
	})
	if err != nil {
		t.Fatalf("updateUserMetronPermissions: %v", err)
	}
	if !output.Body.MetronPermissions.Allowed || output.Body.MetronPermissions.HourlyLimit != 1 {
		t.Fatalf("permissions = %#v; want allowed with hourly limit 1", output.Body.MetronPermissions)
	}

	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/comics"); err != nil {
		t.Fatalf("authorize search: %v", err)
	}
	if err := authorizeMetron(readerCtx, db, metronScopeImport, "POST /metron/comics/{id}/import"); err == nil {
		t.Fatal("authorize import returned nil for search-only user")
	}
	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/series"); err == nil {
		t.Fatal("authorize search returned nil after hourly limit was reached")
	}
}

func TestAdminCanPromoteOtherUsers(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateUserAdmin(adminCtx, db, 2, UpdateUserAdminPayload{IsAdmin: true})
	if err != nil {
		t.Fatalf("updateUserAdmin promote: %v", err)
	}
	if !output.Body.User.IsAdmin {
		t.Fatalf("promoted user isAdmin = false; want true")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updateUserAdmin(readerCtx, db, 1, UpdateUserAdminPayload{IsAdmin: false}); err != nil {
		t.Fatalf("promoted user should be able to update admin roles: %v", err)
	}
	if _, err := updateUserAdmin(readerCtx, db, 2, UpdateUserAdminPayload{IsAdmin: false}); err == nil {
		t.Fatal("updateUserAdmin allowed current user to remove own admin role")
	}
}

func TestListUsersIncludesAccountTimestamps(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, email, email_verified_at, is_admin, created_at)
		VALUES (2, 'Reader', 'reader@example.com', '2026-07-10 11:00:00', 0, '2026-07-09 10:00:00')
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := listUsers(adminCtx, db)
	if err != nil {
		t.Fatalf("listUsers: %v", err)
	}

	for _, entry := range output.Body {
		if entry.User.ID != 2 {
			continue
		}
		if entry.User.CreatedAt != "2026-07-09 10:00:00" {
			t.Fatalf("createdAt = %q; want seeded timestamp", entry.User.CreatedAt)
		}
		if !entry.User.EmailVerified || entry.User.EmailVerifiedAt != "2026-07-10 11:00:00" {
			t.Fatalf("email verification = (%v, %q); want verified timestamp", entry.User.EmailVerified, entry.User.EmailVerifiedAt)
		}
		return
	}

	t.Fatal("reader missing from users response")
}
