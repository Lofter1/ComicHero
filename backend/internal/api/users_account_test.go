package api

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"
)

func TestUpdateAccountRenamesAndRequiresCurrentPassword(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := updateAccount(ctx, db, UpdateAccountPayload{
		Name:            "Renamed",
		CurrentPassword: "wrong",
		NewPassword:     "secret2",
	}); err == nil {
		t.Fatal("updateAccount accepted an incorrect current password")
	}

	output, err := updateAccount(ctx, db, UpdateAccountPayload{
		Name:            "Renamed",
		CurrentPassword: "secret1",
		NewPassword:     "secret2",
	})
	if err != nil {
		t.Fatalf("updateAccount: %v", err)
	}
	if output.Body.User == nil || output.Body.User.Name != "Renamed" {
		t.Fatalf("user = %#v; want renamed current user", output.Body.User)
	}

	var newHash string
	if err := db.Get(&newHash, `SELECT password_hash FROM users WHERE id = 1`); err != nil {
		t.Fatalf("fetch password hash: %v", err)
	}
	if !checkPassword("secret2", newHash) {
		t.Fatal("new password hash does not match updated password")
	}
}

func TestDeleteAccountRequiresPasswordAndAnotherAdmin(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
		INSERT INTO users (id, name, email, password_hash) VALUES (2, 'Other', 'other@example.com', 'hash');
		INSERT INTO user_sessions (token, user_id) VALUES ('session-1', 1);
		INSERT INTO reading_orders (name, author_user_id) VALUES ('Mine', 1);
	`, hash); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "wrong"}); err == nil {
		t.Fatal("deleteAccount accepted an incorrect current password")
	}
	if _, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "secret1"}); err == nil {
		t.Fatal("deleteAccount deleted the only admin account")
	}

	if _, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE id = 2`); err != nil {
		t.Fatalf("promote other account: %v", err)
	}
	output, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "secret1"})
	if err != nil {
		t.Fatalf("deleteAccount: %v", err)
	}
	if output.Body.User != nil || output.Body.Mode != userModeMulti {
		t.Fatalf("status = %#v; want logged-out multi-user status", output.Body)
	}
	if len(output.SetCookie) != 1 || output.SetCookie[0].MaxAge >= 0 {
		t.Fatalf("cookies = %#v; want expired session cookie", output.SetCookie)
	}

	var userCount int
	if err := db.Get(&userCount, `SELECT COUNT(*) FROM users WHERE id = 1`); err != nil {
		t.Fatalf("count deleted user: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("deleted user count = %d; want 0", userCount)
	}
	var sessionCount int
	if err := db.Get(&sessionCount, `SELECT COUNT(*) FROM user_sessions WHERE user_id = 1`); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("deleted user's sessions = %d; want 0", sessionCount)
	}
	var authorCount int
	if err := db.Get(&authorCount, `SELECT COUNT(*) FROM reading_orders WHERE author_user_id = 1`); err != nil {
		t.Fatalf("count authored orders: %v", err)
	}
	if authorCount != 0 {
		t.Fatalf("authored orders = %d; want 0", authorCount)
	}
}

func TestAdminCanDeleteNonAdminUser(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0);
		INSERT INTO user_sessions (token, user_id) VALUES ('reader-session', 2);
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit) VALUES (2, 1, '*', 0);
		INSERT INTO user_metron_request_log (user_id, scope, endpoint) VALUES (2, 'search', '/issue/');
		INSERT INTO user_comics (comic_id, user_id, read) VALUES (1, 2, 1);
		INSERT INTO reading_orders (name, author_user_id) VALUES ('Reader list', 2);
	`); err != nil {
		t.Fatalf("seed reader account: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteUser(adminCtx, db, 2); err != nil {
		t.Fatalf("deleteUser: %v", err)
	}

	for table, query := range map[string]string{
		"users":                   `SELECT COUNT(*) FROM users WHERE id = 2`,
		"user_sessions":           `SELECT COUNT(*) FROM user_sessions WHERE user_id = 2`,
		"user_metron_permissions": `SELECT COUNT(*) FROM user_metron_permissions WHERE user_id = 2`,
		"user_metron_request_log": `SELECT COUNT(*) FROM user_metron_request_log WHERE user_id = 2`,
		"user_comics":             `SELECT COUNT(*) FROM user_comics WHERE user_id = 2`,
	} {
		var count int
		if err := db.Get(&count, query); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d; want 0", table, count)
		}
	}
	var authorCount int
	if err := db.Get(&authorCount, `SELECT COUNT(*) FROM reading_orders WHERE author_user_id = 2`); err != nil {
		t.Fatalf("count reading order authors: %v", err)
	}
	if authorCount != 0 {
		t.Fatalf("reading order author count = %d; want 0", authorCount)
	}
}

func TestNonAdminCannotDeleteUsers(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0);
		INSERT INTO users (id, name, is_admin) VALUES (3, 'Other', 0);
	`); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := deleteUser(readerCtx, db, 3); err == nil {
		t.Fatal("deleteUser by non-admin returned nil error")
	}
}

func TestDeleteUserRejectsOnlyAdmin(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteUser(adminCtx, db, 1); err == nil {
		t.Fatal("deleteUser deleted the only admin account")
	}
}
