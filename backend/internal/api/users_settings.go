package api

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func positivePathID(value string) bool {
	id, err := strconv.Atoi(value)
	return err == nil && id > 0
}

func currentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func userMode(ctx context.Context, db *sqlx.DB) (string, bool, error) {
	var mode string
	if err := db.GetContext(ctx, &mode, `SELECT value FROM app_settings WHERE key = 'user_mode'`); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	if mode != userModeSingle && mode != userModeMulti {
		return "", false, fmt.Errorf("invalid user mode %q", mode)
	}
	return mode, true, nil
}

func registrationMode(ctx context.Context, db *sqlx.DB) (string, error) {
	var mode string
	if err := db.GetContext(ctx, &mode, `SELECT value FROM app_settings WHERE key = 'registration_mode'`); err != nil {
		if err == sql.ErrNoRows {
			return registrationModeInviteOnly, nil
		}
		return "", err
	}
	if mode != registrationModeInviteOnly && mode != registrationModeOpen {
		return "", fmt.Errorf("invalid registration mode %q", mode)
	}
	return mode, nil
}

func publicAccessEnabled(ctx context.Context, db *sqlx.DB) (bool, error) {
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = 'public_access_enabled'`); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return value == "true", nil
}

func getUserStatus(ctx context.Context, db *sqlx.DB, sessionToken string) (*UserStatusOutput, error) {
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
	status := UserStatus{SetupRequired: !configured, Mode: mode, RegistrationMode: regMode, PublicAccess: publicAccess}
	if !configured {
		return &UserStatusOutput{Body: status}, nil
	}

	if mode == userModeSingle {
		userID, err := ensureDefaultUser(ctx, db)
		if err != nil {
			return nil, err
		}
		return userStatusForUser(ctx, db, mode, userID, nil)
	}

	if userID, err := userIDFromSessionToken(ctx, db, sessionToken); err == nil {
		return userStatusForUser(ctx, db, mode, userID, nil)
	}
	return &UserStatusOutput{Body: status}, nil
}
