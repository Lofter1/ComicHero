package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func ensureDefaultUser(ctx context.Context, db sqlx.ExtContext) (int, error) {
	var userID int
	if err := sqlx.GetContext(ctx, db, &userID, `SELECT id FROM users WHERE is_default = 1 OR name = ? ORDER BY is_default DESC, id LIMIT 1`, defaultUserName); err != nil {
		if err != sql.ErrNoRows {
			return 0, huma.Error500InternalServerError("failed to fetch default user")
		}
		result, err := db.ExecContext(ctx, `INSERT INTO users (name, is_default) VALUES (?, 1)`, defaultUserName)
		if err != nil {
			return 0, huma.Error500InternalServerError("failed to create default user")
		}
		id, err := result.LastInsertId()
		if err != nil {
			return 0, huma.Error500InternalServerError("failed to get default user id")
		}
		return int(id), nil
	}
	return userID, nil
}

func getUserByID(ctx context.Context, db *sqlx.DB, id int) (User, error) {
	var user User
	if err := db.GetContext(ctx, &user, `
		SELECT id, name, email, email_verified_at <> '' AS email_verified, is_admin, created_at FROM users WHERE id = ?
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return User{}, huma.Error401Unauthorized("login required")
		}
		return User{}, huma.Error500InternalServerError("failed to fetch user")
	}
	return user, nil
}

func emailVerificationRequiredStatus(ctx context.Context, db *sqlx.DB, email string) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	regMode, err := registrationMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch registration mode")
	}
	publicAccess, err := publicAccessEnabled(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch public access setting")
	}
	return &UserStatusOutput{Body: UserStatus{
		SetupRequired:             !configured,
		Mode:                      mode,
		RegistrationMode:          regMode,
		PublicAccess:              publicAccess,
		EmailVerificationRequired: true,
		EmailVerificationEmail:    email,
	}}, nil
}

func userStatusWithoutUser(ctx context.Context, db *sqlx.DB) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	regMode, err := registrationMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch registration mode")
	}
	publicAccess, err := publicAccessEnabled(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch public access setting")
	}
	return &UserStatusOutput{Body: UserStatus{
		SetupRequired:    !configured,
		Mode:             mode,
		RegistrationMode: regMode,
		PublicAccess:     publicAccess,
	}}, nil
}

func requireAdminUser(ctx context.Context, db *sqlx.DB) (int, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return 0, err
	}
	var isAdmin bool
	if err := db.GetContext(ctx, &isAdmin, `SELECT is_admin FROM users WHERE id = ?`, userID); err != nil {
		if err == sql.ErrNoRows {
			return 0, huma.Error401Unauthorized("login required")
		}
		return 0, huma.Error500InternalServerError("failed to fetch user permissions")
	}
	if !isAdmin {
		return 0, huma.Error403Forbidden("admin access required")
	}
	return userID, nil
}

func userStatusForUser(ctx context.Context, db *sqlx.DB, mode string, userID int, cookie *http.Cookie) (*UserStatusOutput, error) {
	user, err := getUserByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	regMode, err := registrationMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch registration mode")
	}
	publicAccess, err := publicAccessEnabled(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch public access setting")
	}
	metronPermissions, err := metronPermissionsForUser(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	return &UserStatusOutput{
		SetCookie: cookieHeader(cookie),
		Body: UserStatus{
			SetupRequired:     false,
			Mode:              mode,
			RegistrationMode:  regMode,
			PublicAccess:      publicAccess,
			User:              &user,
			MetronPermissions: metronPermissions,
		},
	}, nil
}

func cleanUserName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func cleanEmailAddress(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", fmt.Errorf("valid email is required")
	}
	return email, nil
}

func emailsMatch(email, confirmation string) bool {
	confirmation, err := cleanEmailAddress(confirmation)
	return err == nil && confirmation == email
}
