package api

import (
	"context"
	"database/sql"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func loginUser(ctx context.Context, db *sqlx.DB, payload UserCredentialsPayload) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("login is only available in multi-user mode")
	}

	var row struct {
		ID              int    `db:"id"`
		Email           string `db:"email"`
		EmailVerifiedAt string `db:"email_verified_at"`
		PasswordHash    string `db:"password_hash"`
	}
	email, err := cleanEmailAddress(payload.Email)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	if err := db.GetContext(ctx, &row, `
		SELECT id, email, email_verified_at, password_hash FROM users WHERE email = ?
	`, email); err != nil {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	if !checkPassword(payload.Password, row.PasswordHash) {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	if passwordHashNeedsUpgrade(row.PasswordHash) {
		passwordHash, err := hashPassword(payload.Password)
		if err == nil {
			_, _ = db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, row.ID)
		}
	}
	if row.EmailVerifiedAt == "" {
		return emailVerificationRequiredStatus(ctx, db, row.Email)
	}
	cookie, err := createSession(ctx, db, row.ID)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, row.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to record login")
	}
	return userStatusForUser(ctx, db, mode, row.ID, cookie)
}

func verifyEmail(ctx context.Context, db *sqlx.DB, token string) (*UserStatusOutput, error) {
	userID, err := verifyEmailToken(ctx, db, token)
	if err != nil {
		return nil, err
	}
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("email verification is only available in multi-user mode")
	}
	cookie, err := createSession(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	return userStatusForUser(ctx, db, mode, userID, cookie)
}

func resendEmailVerification(ctx context.Context, db *sqlx.DB, payload ResendEmailVerificationPayload) (*UserStatusOutput, error) {
	email, err := cleanEmailAddress(payload.Email)
	if err != nil {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	var row struct {
		ID              int    `db:"id"`
		Email           string `db:"email"`
		EmailVerifiedAt string `db:"email_verified_at"`
		PasswordHash    string `db:"password_hash"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT id, email, email_verified_at, password_hash
		FROM users
		WHERE email = ?
	`, email); err != nil {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	if !checkPassword(payload.Password, row.PasswordHash) {
		return nil, huma.Error401Unauthorized("invalid email or password")
	}
	if row.EmailVerifiedAt != "" {
		mode, configured, err := userMode(ctx, db)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch user setup")
		}
		if !configured {
			return nil, huma.Error400BadRequest("user setup is not complete")
		}
		return userStatusForUser(ctx, db, mode, row.ID, nil)
	}
	token, _, err := createEmailVerification(ctx, db, row.ID)
	if err != nil {
		return nil, err
	}
	if err := sendEmailVerification(ctx, emailVerificationMessage{
		To:    row.Email,
		Link:  emailVerificationLink(token),
		Token: token,
	}); err != nil {
		return nil, err
	}
	return emailVerificationRequiredStatus(ctx, db, row.Email)
}

func requestPasswordReset(ctx context.Context, db *sqlx.DB, payload ForgotPasswordPayload) (*UserStatusOutput, error) {
	email, err := cleanEmailAddress(payload.Email)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("password reset is only available in multi-user mode")
	}
	var row struct {
		ID    int    `db:"id"`
		Email string `db:"email"`
	}
	if err := db.GetContext(ctx, &row, `SELECT id, email FROM users WHERE email = ?`, email); err != nil {
		if err == sql.ErrNoRows {
			return userStatusWithoutUser(ctx, db)
		}
		return nil, huma.Error500InternalServerError("failed to fetch account")
	}
	token, _, err := createPasswordReset(ctx, db, row.ID)
	if err != nil {
		return nil, err
	}
	if err := sendPasswordReset(ctx, emailVerificationMessage{
		To:    row.Email,
		Link:  passwordResetLink(token),
		Token: token,
	}); err != nil {
		return nil, err
	}
	return userStatusWithoutUser(ctx, db)
}

func resetPassword(ctx context.Context, db *sqlx.DB, payload ResetPasswordPayload) (*UserStatusOutput, error) {
	if len(payload.Password) < 6 {
		return nil, huma.Error400BadRequest("password must be at least 6 characters")
	}
	if payload.PasswordConfirmation != payload.Password {
		return nil, huma.Error400BadRequest("password confirmation must match password")
	}
	userID, err := verifyPasswordResetToken(ctx, db, payload.Token)
	if err != nil {
		return nil, err
	}
	passwordHash, err := hashPassword(payload.Password)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to hash password")
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, email_verified_at = CASE WHEN email_verified_at = '' THEN CURRENT_TIMESTAMP ELSE email_verified_at END
		WHERE id = ?
	`, passwordHash, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to reset password")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = ?`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to clear sessions")
	}
	return userStatusWithoutUser(ctx, db)
}

func logoutUser(ctx context.Context, db *sqlx.DB, token string) (*LogoutUserOutput, error) {
	if token != "" {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token); err != nil {
			return nil, huma.Error500InternalServerError("failed to log out")
		}
	}
	return &LogoutUserOutput{SetCookie: cookieHeader(expiredSessionCookie(ctx))}, nil
}
