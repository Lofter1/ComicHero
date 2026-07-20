package api

import (
	"context"
	"net/http"
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

	emailVerificationRateLimitMaxAttempts = 5
	emailVerificationRateLimitWindow      = time.Minute
	emailVerificationResendMaxAttempts    = 3
	emailVerificationResendWindow         = time.Hour
	passwordResetRequestMaxAttempts       = 3
	passwordResetRequestWindow            = time.Hour
	passwordResetMaxAttempts              = 5
	passwordResetWindow                   = time.Minute
)

type contextUserIDKey struct{}
type contextPublicAccessKey struct{}

func RegisterUserRoutes(api huma.API, db *sqlx.DB) {
	huma.Register(api, huma.Operation{
		OperationID: "listAuditEvents", Tags: []string{tagUsers}, Summary: "List user audit events",
		Description: "Searches, filters, sorts, and paginates successful state-changing API requests. Admin users only.",
		Method:      http.MethodGet, Path: "/audit-events", Errors: []int{401, 403, 500},
	}, func(ctx context.Context, input *AuditEventListInput) (*AuditEventListOutput, error) {
		return listAuditEvents(ctx, db, input)
	})
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
		OperationID: "verifyEmail",
		Tags:        []string{tagUsers},
		Summary:     "Verify email",
		Description: "Verifies a pending email address using a single-use token and starts a session.",
		Method:      http.MethodPost,
		Path:        "/auth/verify-email",
		Errors:      []int{400, 401, 500},
	}, func(ctx context.Context, input *VerifyEmailInput) (*UserStatusOutput, error) {
		return verifyEmail(ctx, db, input.Body.Token)
	})

	huma.Register(api, huma.Operation{
		OperationID: "resendEmailVerification",
		Tags:        []string{tagUsers},
		Summary:     "Resend email verification",
		Description: "Sends a new email verification token after verifying the pending user's password.",
		Method:      http.MethodPost,
		Path:        "/auth/verify-email/resend",
		Errors:      []int{400, 401, 500},
	}, func(ctx context.Context, input *ResendEmailVerificationInput) (*UserStatusOutput, error) {
		return resendEmailVerification(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "requestPasswordReset",
		Tags:        []string{tagUsers},
		Summary:     "Request password reset",
		Description: "Sends a password reset email when the address belongs to a multi-user account.",
		Method:      http.MethodPost,
		Path:        "/auth/forgot-password",
		Errors:      []int{400, 500},
	}, func(ctx context.Context, input *ForgotPasswordInput) (*UserStatusOutput, error) {
		return requestPasswordReset(ctx, db, input.Body)
	})

	huma.Register(api, huma.Operation{
		OperationID: "resetPassword",
		Tags:        []string{tagUsers},
		Summary:     "Reset password",
		Description: "Resets a password using a single-use token sent by email.",
		Method:      http.MethodPost,
		Path:        "/auth/reset-password",
		Errors:      []int{400, 401, 500},
	}, func(ctx context.Context, input *ResetPasswordInput) (*UserStatusOutput, error) {
		return resetPassword(ctx, db, input.Body)
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
