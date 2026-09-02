package api

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSwitchToMultiUserRequiresSingleUserSetup(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)

	// No mode configured yet.
	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted unconfigured setup")
	}

	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}
	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted an instance already in multi-user mode")
	}
}

func TestSwitchToMultiUserRequiresAdmin(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'single');
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Guest', 0);
	`); err != nil {
		t.Fatalf("seed single-user mode: %v", err)
	}

	guestCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := switchToMultiUser(guestCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser allowed a non-admin user to switch modes")
	}
}

func TestSwitchToMultiUserValidatesInput(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'single')`); err != nil {
		t.Fatalf("seed single-user mode: %v", err)
	}
	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)

	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "",
		Email:    "justin@example.com",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted a blank name")
	}
	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "not-an-email",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted an invalid email")
	}
	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "short",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted a too-short password")
	}
}

func TestSwitchToMultiUserUpgradesInstance(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'single')`); err != nil {
		t.Fatalf("seed single-user mode: %v", err)
	}
	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)

	status, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "secret1",
	})
	if err != nil {
		t.Fatalf("switchToMultiUser: %v", err)
	}
	if status.Body.Mode != userModeMulti {
		t.Fatalf("mode = %q; want %q", status.Body.Mode, userModeMulti)
	}
	if status.Body.User == nil || status.Body.User.Name != "Justin" || status.Body.User.Email != "justin@example.com" {
		t.Fatalf("user = %#v; want updated name and email", status.Body.User)
	}
	if !status.Body.User.EmailVerified {
		t.Fatal("switchToMultiUser did not mark the email as verified")
	}
	if len(status.SetCookie) == 0 {
		t.Fatal("switchToMultiUser did not start a session")
	}

	mode, configured, err := userMode(context.Background(), db)
	if err != nil || !configured || mode != userModeMulti {
		t.Fatalf("userMode = (%q, %v, %v); want (%q, true, nil)", mode, configured, err, userModeMulti)
	}

	// Switching again is rejected now that the instance is already multi-user.
	if _, err := switchToMultiUser(adminCtx, db, SwitchToMultiUserPayload{
		Name:     "Justin",
		Email:    "justin@example.com",
		Password: "secret1",
	}); err == nil {
		t.Fatal("switchToMultiUser accepted a second switch")
	}
}
