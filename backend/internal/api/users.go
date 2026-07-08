package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const (
	userModeSingle = "single"
	userModeMulti  = "multi"

	registrationModeInviteOnly = "invite_only"
	registrationModeOpen       = "open"

	sessionCookieName = "comichero_session"
	defaultUserName   = "Default"

	userInviteTokenBytes = 32
	userInviteTTL        = 7 * 24 * time.Hour
	sessionTTL           = 30 * 24 * time.Hour

	loginRateLimitMaxAttempts = 5
	loginRateLimitWindow      = time.Minute

	registrationRateLimitMaxAttempts = 3
	registrationRateLimitWindow      = time.Hour
)

type contextUserIDKey struct{}
type contextPublicAccessKey struct{}

func RegisterUserRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "getUserStatus",
		Tags:        []string{tagUsers},
		Summary:     "Get user setup and session status",
		Description: "Returns whether ComicHero has been configured for single-user or multi-user use and, when logged in, the current user.",
		Method:      http.MethodGet,
		Path:        "/auth/status",
		Errors:      errsRead,
	}, func(ctx context.Context, input *UserStatusInput) (*UserStatusOutput, error) {
		return getUserStatus(ctx, db, input.Session)
	})

	huma.Register(api, huma.Operation{
		OperationID: "setupUsers",
		Tags:        []string{tagUsers},
		Summary:     "Choose user mode",
		Description: "Configures ComicHero for single-user use without login or multi-user use with login. Existing read status remains attached to the initial user.",
		Method:      http.MethodPost,
		Path:        "/auth/setup",
		Errors:      []int{400, 409, 422, 500},
	}, func(ctx context.Context, input *SetupUsersInput) (*UserStatusOutput, error) {
		return setupUsers(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "registerUser",
		Tags:        []string{tagUsers},
		Summary:     "Register a user",
		Description: "Creates a new user in multi-user mode and signs them in.",
		Method:      http.MethodPost,
		Path:        "/auth/register",
		Errors:      []int{400, 409, 422, 500},
	}, func(ctx context.Context, input *RegisterUserInput) (*UserStatusOutput, error) {
		return registerUser(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "loginUser",
		Tags:        []string{tagUsers},
		Summary:     "Log in",
		Description: "Starts a multi-user session.",
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Errors:      []int{400, 401, 500},
	}, func(ctx context.Context, input *LoginUserInput) (*UserStatusOutput, error) {
		return loginUser(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "logoutUser",
		Tags:          []string{tagUsers},
		Summary:       "Log out",
		Description:   "Ends the current multi-user session.",
		Method:        http.MethodPost,
		Path:          "/auth/logout",
		DefaultStatus: 204,
		Errors:        []int{500},
	}, func(ctx context.Context, input *LogoutUserInput) (*LogoutUserOutput, error) {
		return logoutUser(ctx, db, input.Session)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateAccount",
		Tags:        []string{tagUsers},
		Summary:     "Update current account",
		Description: "Updates the current user's display name and, in multi-user mode, optionally changes the password after verifying the current password.",
		Method:      http.MethodPut,
		Path:        "/account",
		Errors:      []int{400, 401, 409, 422, 500},
	}, func(ctx context.Context, input *UpdateAccountInput) (*UserStatusOutput, error) {
		return updateAccount(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteAccount",
		Tags:        []string{tagUsers},
		Summary:     "Delete current account",
		Description: "Deletes the current multi-user account after verifying the current password. The final user or final admin account cannot be deleted.",
		Method:      http.MethodDelete,
		Path:        "/account",
		Errors:      []int{400, 401, 500},
	}, func(ctx context.Context, input *DeleteAccountInput) (*UserStatusOutput, error) {
		return deleteAccount(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listUsers",
		Tags:        []string{tagUsers},
		Summary:     "List users",
		Description: "Lists users and their Metron permissions. Admin users only.",
		Method:      http.MethodGet,
		Path:        "/users",
		Errors:      []int{401, 403, 500},
	}, func(ctx context.Context, input *struct{}) (*UserListOutput, error) {
		return listUsers(ctx, db)
	})

	huma.Register(api, huma.Operation{
		OperationID: "createUserInvite",
		Tags:        []string{tagUsers},
		Summary:     "Create user invite",
		Description: "Creates a single-use registration invite. Admin users only.",
		Method:      http.MethodPost,
		Path:        "/users/invites",
		Errors:      []int{401, 403, 500},
	}, func(ctx context.Context, input *struct{}) (*UserInviteOutput, error) {
		return createUserInvite(ctx, db)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateRegistrationMode",
		Tags:        []string{tagUsers},
		Summary:     "Update registration mode",
		Description: "Controls whether registration requires invite tokens or is open. Admin users only.",
		Method:      http.MethodPut,
		Path:        "/users/registration-mode",
		Errors:      []int{400, 401, 403, 500},
	}, func(ctx context.Context, input *UpdateRegistrationModeInput) (*RegistrationModeOutput, error) {
		return updateRegistrationMode(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updatePublicAccess",
		Tags:        []string{tagUsers},
		Summary:     "Update public read access",
		Description: "Controls whether anonymous visitors can browse read-only library data and export reading orders as CBL. Admin users only.",
		Method:      http.MethodPut,
		Path:        "/users/public-access",
		Errors:      []int{400, 401, 403, 500},
	}, func(ctx context.Context, input *UpdatePublicAccessInput) (*PublicAccessOutput, error) {
		return updatePublicAccess(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateUserMetronPermissions",
		Tags:        []string{tagUsers},
		Summary:     "Update user Metron permissions",
		Description: "Controls whether a user can call Metron endpoints, which endpoint scopes are allowed, and the per-hour endpoint limit. Admin users only.",
		Method:      http.MethodPut,
		Path:        "/users/{id}/metron-permissions",
		Errors:      []int{400, 401, 403, 404, 500},
	}, func(ctx context.Context, input *UpdateUserMetronPermissionsInput) (*UserAdminOutput, error) {
		return updateUserMetronPermissions(ctx, db, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateUserAdmin",
		Tags:        []string{tagUsers},
		Summary:     "Update user admin role",
		Description: "Promotes or demotes a user account. Admin users only.",
		Method:      http.MethodPut,
		Path:        "/users/{id}/admin",
		Errors:      []int{400, 401, 403, 404, 500},
	}, func(ctx context.Context, input *UpdateUserAdminInput) (*UserAdminOutput, error) {
		return updateUserAdmin(ctx, db, input.ID, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "deleteUser",
		Tags:        []string{tagUsers},
		Summary:     "Delete user account",
		Description: "Deletes a user account and its per-user data. Admin users only. The final admin account cannot be deleted.",
		Method:      http.MethodDelete,
		Path:        "/users/{id}",
		Errors:      []int{400, 401, 403, 404, 409, 500},
	}, func(ctx context.Context, input *DeleteUserInput) (*struct{}, error) {
		return deleteUser(ctx, db, input.ID)
	})
}

func UserMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isLoginRequest(r) && !authLoginLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many login attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isRegisterRequest(r) && !authRegistrationLimiter.allow(clientIP(r), time.Now()) {
				http.Error(w, "too many registration attempts, try again later", http.StatusTooManyRequests)
				return
			}
			if isUserRouteAllowedWithoutSession(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			mode, configured, err := userMode(r.Context(), db)
			if err != nil {
				http.Error(w, "failed to read user setup", http.StatusInternalServerError)
				return
			}
			if !configured {
				http.Error(w, "user setup required", http.StatusPreconditionRequired)
				return
			}

			var userID int
			publicAccess := false
			if mode == userModeSingle {
				userID, err = ensureDefaultUser(r.Context(), db)
			} else {
				userID, err = sessionUserID(r, db)
				if err != nil && isPublicReadRequest(r) {
					enabled, settingErr := publicAccessEnabled(r.Context(), db)
					if settingErr != nil {
						http.Error(w, "failed to read public access setting", http.StatusInternalServerError)
						return
					}
					if enabled {
						userID, err = ensureDefaultUser(r.Context(), db)
						publicAccess = err == nil
					}
				}
			}
			if err != nil {
				http.Error(w, "login required", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextUserIDKey{}, userID)
			if publicAccess {
				ctx = context.WithValue(ctx, contextPublicAccessKey{}, true)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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
		user, err := getUserByID(ctx, db, userID)
		if err != nil {
			return nil, err
		}
		status.User = &user
		return &UserStatusOutput{Body: status}, nil
	}

	if userID, err := userIDFromSessionToken(ctx, db, sessionToken); err == nil {
		user, err := getUserByID(ctx, db, userID)
		if err != nil {
			return nil, err
		}
		status.User = &user
	}
	return &UserStatusOutput{Body: status}, nil
}

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
		if len(payload.Password) < 6 {
			return nil, huma.Error400BadRequest("password must be at least 6 characters")
		}
		passwordHash, err := hashPassword(payload.Password)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to hash password")
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE users
			SET name = ?, password_hash = ?, is_default = 0
			WHERE id = ?
		`, name, passwordHash, userID); err != nil {
			return nil, huma.Error409Conflict("user name already exists")
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
	if len(payload.Password) < 6 {
		return nil, huma.Error400BadRequest("password must be at least 6 characters")
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

	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (name, password_hash)
		VALUES (?, ?)
	`, name, passwordHash)
	if err != nil {
		return nil, huma.Error409Conflict("user name already exists")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get user id")
	}
	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to create user")
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

func loginUser(ctx context.Context, db *sqlx.DB, payload UserCredentialsPayload) (*UserStatusOutput, error) {
	mode, configured, err := userMode(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch user setup")
	}
	if !configured || mode != userModeMulti {
		return nil, huma.Error400BadRequest("login is only available in multi-user mode")
	}

	var row struct {
		ID           int    `db:"id"`
		PasswordHash string `db:"password_hash"`
	}
	if err := db.GetContext(ctx, &row, `
		SELECT id, password_hash FROM users WHERE name = ?
	`, cleanUserName(payload.Name)); err != nil {
		return nil, huma.Error401Unauthorized("invalid user name or password")
	}
	if !checkPassword(payload.Password, row.PasswordHash) {
		return nil, huma.Error401Unauthorized("invalid user name or password")
	}
	if passwordHashNeedsUpgrade(row.PasswordHash) {
		passwordHash, err := hashPassword(payload.Password)
		if err == nil {
			_, _ = db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, row.ID)
		}
	}
	cookie, err := createSession(ctx, db, row.ID)
	if err != nil {
		return nil, err
	}
	return userStatusForUser(ctx, db, mode, row.ID, cookie)
}

func logoutUser(ctx context.Context, db *sqlx.DB, token string) (*LogoutUserOutput, error) {
	if token != "" {
		if _, err := db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token = ?`, token); err != nil {
			return nil, huma.Error500InternalServerError("failed to log out")
		}
	}
	return &LogoutUserOutput{SetCookie: cookieHeader(expiredSessionCookie())}, nil
}

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
		SetCookie: cookieHeader(expiredSessionCookie()),
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
	defer tx.Rollback()

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
		SELECT id, name, is_admin FROM users WHERE id = ?
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return User{}, huma.Error401Unauthorized("login required")
		}
		return User{}, huma.Error500InternalServerError("failed to fetch user")
	}
	return user, nil
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
