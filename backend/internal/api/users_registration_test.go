package api

import (
	"context"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRegisterUserRequiresValidInvite(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}

	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "No Invite",
		Email:                "no-invite@example.com",
		EmailConfirmation:    "no-invite@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	invite, err := createUserInvite(adminCtx, db)
	if err != nil {
		t.Fatalf("createUserInvite: %v", err)
	}
	if invite.Body.Token == "" || invite.Body.ExpiresAt == "" {
		t.Fatalf("invite = %#v; want token and expiry", invite.Body)
	}

	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Missing Email Confirmation",
		Email:                "missing-email-confirmation@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted missing email confirmation")
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Password Mismatch",
		Email:                "password-mismatch@example.com",
		EmailConfirmation:    "password-mismatch@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret2",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted mismatched password confirmation")
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Invited",
		Email:                "invited@example.com",
		EmailConfirmation:    "invited@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err != nil {
		t.Fatalf("registerUser with invite: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Reuse",
		Email:                "reuse@example.com",
		EmailConfirmation:    "reuse@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted a used invite")
	}

	if _, err := db.Exec(`
		INSERT INTO user_invites (token, expires_at)
		VALUES ('expired-token', ?)
	`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed expired invite: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Expired",
		Email:                "expired@example.com",
		EmailConfirmation:    "expired@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          "expired-token",
	}); err == nil {
		t.Fatal("registerUser accepted an expired invite")
	}
}

func TestRegistrationModeDefaultsAndAdminCanUpdate(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)`); err != nil {
		t.Fatalf("seed reader user: %v", err)
	}

	mode, err := registrationMode(context.Background(), db)
	if err != nil {
		t.Fatalf("registrationMode default: %v", err)
	}
	if mode != registrationModeInviteOnly {
		t.Fatalf("registrationMode default = %q; want %q", mode, registrationModeInviteOnly)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Default Blocked",
		Email:                "default-blocked@example.com",
		EmailConfirmation:    "default-blocked@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite succeeded while registration mode is unset")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updateRegistrationMode(readerCtx, db, UpdateRegistrationModePayload{Mode: registrationModeOpen}); err == nil {
		t.Fatal("non-admin updateRegistrationMode returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateRegistrationMode(adminCtx, db, UpdateRegistrationModePayload{Mode: registrationModeOpen})
	if err != nil {
		t.Fatalf("updateRegistrationMode: %v", err)
	}
	if output.Body.RegistrationMode != registrationModeOpen {
		t.Fatalf("output registrationMode = %q; want %q", output.Body.RegistrationMode, registrationModeOpen)
	}
	mode, err = registrationMode(context.Background(), db)
	if err != nil {
		t.Fatalf("registrationMode after update: %v", err)
	}
	if mode != registrationModeOpen {
		t.Fatalf("registrationMode after update = %q; want %q", mode, registrationModeOpen)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Signup",
		Email:                "open-signup@example.com",
		EmailConfirmation:    "open-signup@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err != nil {
		t.Fatalf("registerUser without invite in open mode: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Mismatch",
		Email:                "open-mismatch@example.com",
		EmailConfirmation:    "other@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser accepted mismatched email confirmation in open mode")
	}

	if _, err := updateRegistrationMode(adminCtx, db, UpdateRegistrationModePayload{Mode: registrationModeInviteOnly}); err != nil {
		t.Fatalf("updateRegistrationMode back to invite_only: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Invite Blocked",
		Email:                "invite-blocked@example.com",
		EmailConfirmation:    "invite-blocked@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite succeeded after switching back to invite_only")
	}
}

func TestOpenRegistrationRequiresEmailVerificationBeforeAccess(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO app_settings (key, value) VALUES ('registration_mode', 'open');
	`); err != nil {
		t.Fatalf("seed open registration mode: %v", err)
	}

	output, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Pending",
		Email:                "open-pending@example.com",
		EmailConfirmation:    "open-pending@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	})
	if err != nil {
		t.Fatalf("registerUser open: %v", err)
	}
	if !output.Body.EmailVerificationRequired || output.Body.EmailVerificationEmail != "open-pending@example.com" {
		t.Fatalf("registration status = %#v; want email verification required", output.Body)
	}
	if len(output.SetCookie) != 0 {
		t.Fatalf("registration cookies = %#v; want no session before verification", output.SetCookie)
	}

	var row struct {
		ID              int    `db:"id"`
		EmailVerifiedAt string `db:"email_verified_at"`
	}
	if err := db.Get(&row, `SELECT id, email_verified_at FROM users WHERE email = 'open-pending@example.com'`); err != nil {
		t.Fatalf("fetch registered user: %v", err)
	}
	if row.EmailVerifiedAt != "" {
		t.Fatalf("email_verified_at = %q; want empty before verification", row.EmailVerifiedAt)
	}
	var tokenCount int
	if err := db.Get(&tokenCount, `SELECT COUNT(*) FROM user_email_verifications WHERE user_id = ? AND used_at = ''`, row.ID); err != nil {
		t.Fatalf("count verification tokens: %v", err)
	}
	if tokenCount != 1 {
		t.Fatalf("active verification token count = %d; want 1", tokenCount)
	}

	loginOutput, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Email:    "open-pending@example.com",
		Password: "secret1",
	})
	if err != nil {
		t.Fatalf("loginUser pending: %v", err)
	}
	if !loginOutput.Body.EmailVerificationRequired || len(loginOutput.SetCookie) != 0 {
		t.Fatalf("pending login = %#v cookies %#v; want verification required without session", loginOutput.Body, loginOutput.SetCookie)
	}

	token, _, err := createEmailVerification(context.Background(), db, row.ID)
	if err != nil {
		t.Fatalf("create verification token: %v", err)
	}
	verified, err := verifyEmail(context.Background(), db, token)
	if err != nil {
		t.Fatalf("verifyEmail: %v", err)
	}
	if verified.Body.User == nil || !verified.Body.User.EmailVerified {
		t.Fatalf("verified user = %#v; want verified session user", verified.Body.User)
	}
	if len(verified.SetCookie) != 1 || verified.SetCookie[0].Value == "" {
		t.Fatalf("verification cookies = %#v; want session", verified.SetCookie)
	}
}

func TestPasswordResetChangesPasswordAndConsumesToken(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
		INSERT INTO user_sessions (token, user_id, expires_at) VALUES ('old-session', 1, '2999-01-01T00:00:00Z');
	`, hash); err != nil {
		t.Fatalf("seed reset user: %v", err)
	}

	if _, err := requestPasswordReset(context.Background(), db, ForgotPasswordPayload{Email: "test@example.com"}); err != nil {
		t.Fatalf("requestPasswordReset: %v", err)
	}
	var userID int
	if err := db.Get(&userID, `SELECT id FROM users WHERE email = 'test@example.com'`); err != nil {
		t.Fatalf("fetch user id: %v", err)
	}
	var tokenCount int
	if err := db.Get(&tokenCount, `SELECT COUNT(*) FROM user_password_resets WHERE user_id = ? AND used_at = ''`, userID); err != nil {
		t.Fatalf("count reset tokens: %v", err)
	}
	if tokenCount != 1 {
		t.Fatalf("active reset token count = %d; want 1", tokenCount)
	}

	token, _, err := createPasswordReset(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("createPasswordReset: %v", err)
	}
	if _, err := resetPassword(context.Background(), db, ResetPasswordPayload{
		Token:                token,
		Password:             "secret2",
		PasswordConfirmation: "secret2",
	}); err != nil {
		t.Fatalf("resetPassword: %v", err)
	}
	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{Email: "test@example.com", Password: "secret1"}); err == nil {
		t.Fatal("loginUser accepted old password after reset")
	}
	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{Email: "test@example.com", Password: "secret2"}); err != nil {
		t.Fatalf("loginUser with new password: %v", err)
	}
	if _, err := resetPassword(context.Background(), db, ResetPasswordPayload{
		Token:                token,
		Password:             "secret3",
		PasswordConfirmation: "secret3",
	}); err == nil {
		t.Fatal("resetPassword accepted a used token")
	}
	var sessionCount int
	if err := db.Get(&sessionCount, `SELECT COUNT(*) FROM user_sessions WHERE token = 'old-session'`); err != nil {
		t.Fatalf("count old sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("old session count = %d; want 0", sessionCount)
	}
}
