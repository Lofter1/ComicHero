package api

import (
	"context"
	"database/sql"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func updateAccount(ctx context.Context, db *sqlx.DB, payload UpdateAccountPayload) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured {
		return nil, huma.Error400BadRequest("user setup is not complete")
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := cleanUserName(payload.Name)
	if name == "" {
		return nil, huma.Error400BadRequest("name is required")
	}

	newPassword := strings.TrimSpace(payload.NewPassword)
	if newPassword != "" && len(payload.NewPassword) < 6 {
		return nil, huma.Error400BadRequest("new password must be at least 6 characters")
	}
	if newPassword != "" && mode != userModeMulti {
		return nil, huma.Error400BadRequest("passwords are only used in multi-user mode")
	}

	var passwordHash string
	if newPassword != "" {
		var currentHash string
		if err := db.GetContext(ctx, &currentHash, `SELECT password_hash FROM users WHERE id = ?`, userID); err != nil {
			if err == sql.ErrNoRows {
				return nil, huma.Error401Unauthorized("login required")
			}
			return nil, huma.Error500InternalServerError("failed to fetch account")
		}
		if !checkPassword(payload.CurrentPassword, currentHash) {
			return nil, huma.Error401Unauthorized("current password is incorrect")
		}
		passwordHash, err = hashPassword(payload.NewPassword)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to hash password")
		}
	}

	if passwordHash == "" {
		if _, err := db.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, name, userID); err != nil {
			return nil, huma.Error409Conflict("user name already exists")
		}
	} else if _, err := db.ExecContext(ctx, `UPDATE users SET name = ?, password_hash = ? WHERE id = ?`, name, passwordHash, userID); err != nil {
		return nil, huma.Error409Conflict("user name already exists")
	}

	return userStatusForUser(ctx, db, mode, userID, nil)
}

func deleteAccount(ctx context.Context, db *sqlx.DB, payload DeleteAccountPayload) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("account deletion is only available in multi-user mode")
	}

	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	var row struct {
		PasswordHash string `db:"password_hash"`
		IsAdmin      bool   `db:"is_admin"`
	}
	if err := db.GetContext(ctx, &row, `SELECT password_hash, is_admin FROM users WHERE id = ?`, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, huma.Error401Unauthorized("login required")
		}
		return nil, huma.Error500InternalServerError("failed to fetch account")
	}
	if !checkPassword(payload.CurrentPassword, row.PasswordHash) {
		return nil, huma.Error401Unauthorized("current password is incorrect")
	}

	var userCount int
	if err := db.GetContext(ctx, &userCount, `SELECT COUNT(*) FROM users`); err != nil {
		return nil, huma.Error500InternalServerError("failed to count users")
	}
	if userCount <= 1 {
		return nil, huma.Error400BadRequest("cannot delete the only user account")
	}

	if row.IsAdmin {
		var adminCount int
		if err := db.GetContext(ctx, &adminCount, `SELECT COUNT(*) FROM users WHERE is_admin = 1`); err != nil {
			return nil, huma.Error500InternalServerError("failed to count admins")
		}
		if adminCount <= 1 {
			return nil, huma.Error400BadRequest("cannot delete the only admin account")
		}
	}

	if err := deleteUserData(ctx, db, userID); err != nil {
		return nil, err
	}
	publicAccess, err := publicAccessEnabled(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch public access setting")
	}

	return &UserStatusOutput{
		SetCookie: cookieHeader(expiredSessionCookie(ctx)),
		Body: UserStatus{
			SetupRequired: false,
			Mode:          mode,
			PublicAccess:  publicAccess,
		},
	}, nil
}

func deleteUser(ctx context.Context, db *sqlx.DB, userID int) (*struct{}, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, huma.Error400BadRequest("user id is required")
	}

	var isAdmin bool
	if err := db.GetContext(ctx, &isAdmin, `SELECT is_admin FROM users WHERE id = ?`, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, huma.Error404NotFound("user not found")
		}
		return nil, huma.Error500InternalServerError("failed to fetch user")
	}
	if isAdmin {
		var adminCount int
		if err := db.GetContext(ctx, &adminCount, `SELECT COUNT(*) FROM users WHERE is_admin = 1`); err != nil {
			return nil, huma.Error500InternalServerError("failed to count admins")
		}
		if adminCount <= 1 {
			return nil, huma.Error409Conflict("cannot remove the last admin")
		}
	}

	if err := deleteUserData(ctx, db, userID); err != nil {
		return nil, err
	}
	return &struct{}{}, nil
}

func deleteUserData(ctx context.Context, db *sqlx.DB, userID int) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return huma.Error500InternalServerError("failed to start account deletion")
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to delete sessions")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_metron_request_log WHERE user_id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to delete Metron request history")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_metron_permissions WHERE user_id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to delete Metron permissions")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_comics WHERE user_id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to delete read status")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE reading_orders SET author_user_id = NULL WHERE author_user_id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to clear reading order authorship")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		return huma.Error500InternalServerError("failed to delete account")
	}
	if err := tx.Commit(); err != nil {
		return huma.Error500InternalServerError("failed to delete account")
	}
	return nil
}
