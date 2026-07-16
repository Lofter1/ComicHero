package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func setupUsers(ctx context.Context, db *sqlx.DB, payload SetupUsersPayload) (*UserStatusOutput, error) {
	mode := strings.TrimSpace(payload.Mode)
	if mode != userModeSingle && mode != userModeMulti {
		return nil, huma.Error400BadRequest("mode must be single or multi")
	}
	if _, configured, err := userMode(ctx, db); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	} else if configured {
		return nil, huma.Error409Conflict("user setup is already complete")
	}

	userID, err := ensureDefaultUser(ctx, db)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start user setup")
	}
	defer tx.Rollback()

	if mode == userModeMulti {
		name := cleanUserName(payload.Name)
		if name == "" {
			return nil, huma.Error400BadRequest("name is required for multi-user setup")
		}
		email, err := cleanEmailAddress(payload.Email)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if len(payload.Password) < 6 {
			return nil, huma.Error400BadRequest("password must be at least 6 characters")
		}
		passwordHash, err := hashPassword(payload.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to hash password")
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE users
			SET name = ?, email = ?, email_verified_at = CURRENT_TIMESTAMP, password_hash = ?, is_default = 0
			WHERE id = ?
		`, name, email, passwordHash, userID); err != nil {
			return nil, huma.Error409Conflict("user name or email already exists")
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('user_mode', ?)
	`, mode); err != nil {
		return nil, huma.Error500InternalServerError("failed to save user setup")
	}
	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to save user setup")
	}

	var cookie *http.Cookie
	if mode == userModeMulti {
		cookie, err = createSession(ctx, db, userID)
		if err != nil {
			return nil, err
		}
	}
	return userStatusForUser(ctx, db, mode, userID, cookie)
}

func registerUser(ctx context.Context, db *sqlx.DB, payload UserCredentialsPayload) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("registration is only available in multi-user mode")
	}
	regMode, err := registrationMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch registration mode")
	}
	name := cleanUserName(payload.Name)
	if name == "" {
		return nil, huma.Error400BadRequest("name is required")
	}
	email, err := cleanEmailAddress(payload.Email)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	if !emailsMatch(email, payload.EmailConfirmation) {
		return nil, huma.Error400BadRequest("email confirmation must match email")
	}
	if len(payload.Password) < 6 {
		return nil, huma.Error400BadRequest("password must be at least 6 characters")
	}
	if payload.PasswordConfirmation != payload.Password {
		return nil, huma.Error400BadRequest("password confirmation must match password")
	}
	passwordHash, err := hashPassword(payload.Password)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to hash password")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start registration")
	}
	defer tx.Rollback()

	if regMode == registrationModeInviteOnly {
		if err := consumeUserInvite(ctx, tx, payload.InviteToken); err != nil {
			return nil, err
		}
	}

	verifiedAt := currentTimestamp()
	if requiresEmailVerification(regMode) {
		verifiedAt = ""
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (name, email, email_verified_at, password_hash)
		VALUES (?, ?, ?, ?)
	`, name, email, verifiedAt, passwordHash)
	if err != nil {
		return nil, huma.Error409Conflict("user name or email already exists")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get user id")
	}
	var verificationToken string
	if requiresEmailVerification(regMode) {
		token, _, err := createEmailVerification(ctx, tx, int(id))
		if err != nil {
			return nil, err
		}
		verificationToken = token
	}
	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to create user")
	}
	if verificationToken != "" {
		if err := sendEmailVerification(ctx, emailVerificationMessage{
			To:    email,
			Link:  emailVerificationLink(verificationToken),
			Token: verificationToken,
		}); err != nil {
			return nil, err
		}
		return emailVerificationRequiredStatus(ctx, db, email)
	}
	cookie, err := createSession(ctx, db, int(id))
	if err != nil {
		return nil, err
	}
	return userStatusForUser(ctx, db, mode, int(id), cookie)
}

func createUserInvite(ctx context.Context, db *sqlx.DB) (*UserInviteOutput, error) {
	userID, err := requireAdminUser(ctx, db)
	if err != nil {
		return nil, err
	}
	token, err := randomToken(userInviteTokenBytes)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create invite token")
	}
	expiresAt := time.Now().UTC().Add(userInviteTTL).Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_invites (token, created_by_user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, expiresAt); err != nil {
		return nil, huma.Error500InternalServerError("failed to save invite")
	}
	return &UserInviteOutput{Body: UserInvite{Token: token, ExpiresAt: expiresAt}}, nil
}

func updateRegistrationMode(ctx context.Context, db *sqlx.DB, payload UpdateRegistrationModePayload) (*RegistrationModeOutput, error) {
	userID, err := requireAdminUser(ctx, db)
	if err != nil {
		return nil, err
	}
	mode := strings.TrimSpace(payload.Mode)
	if mode != registrationModeInviteOnly && mode != registrationModeOpen {
		return nil, huma.Error400BadRequest("mode must be invite_only or open")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('registration_mode', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, mode); err != nil {
		return nil, huma.Error500InternalServerError("failed to save registration mode")
	}
	currentMode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured {
		return nil, huma.Error400BadRequest("user setup is not complete")
	}
	status, err := userStatusForUser(ctx, db, currentMode, userID, nil)
	if err != nil {
		return nil, err
	}
	status.Body.RegistrationMode = mode
	return &RegistrationModeOutput{Body: status.Body}, nil
}

func updatePublicAccess(ctx context.Context, db *sqlx.DB, payload UpdatePublicAccessPayload) (*PublicAccessOutput, error) {
	userID, err := requireAdminUser(ctx, db)
	if err != nil {
		return nil, err
	}
	value := "false"
	if payload.Enabled {
		value = "true"
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('public_access_enabled', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, value); err != nil {
		return nil, huma.Error500InternalServerError("failed to save public access setting")
	}
	currentMode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured {
		return nil, huma.Error400BadRequest("user setup is not complete")
	}
	status, err := userStatusForUser(ctx, db, currentMode, userID, nil)
	if err != nil {
		return nil, err
	}
	status.Body.PublicAccess = payload.Enabled
	return &PublicAccessOutput{Body: status.Body}, nil
}

func consumeUserInvite(ctx context.Context, db sqlx.ExtContext, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return huma.Error401Unauthorized("valid invite token is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.ExecContext(ctx, `
		UPDATE user_invites
		SET used_at = ?
		WHERE token = ?
		  AND used_at = ''
		  AND (expires_at = '' OR expires_at > ?)
	`, now, token, now)
	if err != nil {
		return huma.Error500InternalServerError("failed to consume invite")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return huma.Error500InternalServerError("failed to check invite")
	}
	if rows == 0 {
		return huma.Error401Unauthorized("valid invite token is required")
	}
	return nil
}
