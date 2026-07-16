package api

import "net/http"

type User struct {
	ID              int    `json:"id"              db:"id"                doc:"Local user identifier." example:"1"`
	Name            string `json:"name"            db:"name"              doc:"Display name." example:"Justin"`
	Email           string `json:"email"           db:"email"             doc:"Email address used to log in." example:"reader@example.com"`
	EmailVerified   bool   `json:"emailVerified"   db:"email_verified"    doc:"Whether this user's email address has been verified." example:"true"`
	EmailVerifiedAt string `json:"emailVerifiedAt" db:"email_verified_at" doc:"When the user's email address was verified." example:"2026-07-10T10:30:00Z"`
	IsAdmin         bool   `json:"isAdmin"         db:"is_admin"          doc:"Whether the user can manage user permissions." example:"false"`
	CreatedAt       string `json:"createdAt"       db:"created_at"        doc:"When the user account was created." example:"2026-07-10T10:00:00Z"`
	LastLoginAt     string `json:"lastLoginAt"     db:"last_login_at"     doc:"When the user most recently logged in successfully." example:"2026-07-10T11:00:00Z"`
}

type UserMetronPermissions struct {
	Allowed     bool     `json:"allowed"     doc:"Whether this user can call ComicHero Metron endpoints." example:"true"`
	Scopes      []string `json:"scopes"      doc:"Allowed Metron scopes. Use * for all, or combine search, detail, import, and monitor." example:"search"`
	HourlyLimit int      `json:"hourlyLimit" minimum:"0" doc:"Maximum Metron endpoint calls per rolling hour. Use 0 for unlimited." example:"60"`
}

type UserAdminView struct {
	User              User                  `json:"user"              doc:"User account."`
	MetronPermissions UserMetronPermissions `json:"metronPermissions" doc:"Metron endpoint permissions for this user."`
}

type UserListOutput struct {
	Body []UserAdminView
}

type UserInvite struct {
	Token     string `json:"token"     doc:"Single-use invite token."`
	ExpiresAt string `json:"expiresAt" doc:"RFC3339 expiry time for this invite."`
}

type UserInviteOutput struct {
	Body UserInvite
}

type UpdateUserMetronPermissionsInput struct {
	ID   int `path:"id" doc:"Local user identifier." example:"2"`
	Body UserMetronPermissions
}

type UpdateUserAdminPayload struct {
	IsAdmin bool `json:"isAdmin" doc:"Whether the user should be an admin." example:"true"`
}

type UpdateUserAdminInput struct {
	ID   int `path:"id" doc:"Local user identifier." example:"2"`
	Body UpdateUserAdminPayload
}

type DeleteUserInput struct {
	ID int `path:"id" doc:"Local user identifier." example:"2"`
}

type UserAdminOutput struct {
	Body UserAdminView
}

type UpdateRegistrationModePayload struct {
	Mode string `json:"mode" doc:"Registration mode: invite_only requires invite tokens, open allows self-registration." enum:"invite_only,open" example:"invite_only"`
}

type UpdateRegistrationModeInput struct {
	Body UpdateRegistrationModePayload
}

type RegistrationModeOutput struct {
	Body UserStatus
}

type UpdatePublicAccessPayload struct {
	Enabled bool `json:"enabled" doc:"Whether anonymous read-only public access is enabled." example:"true"`
}

type UpdatePublicAccessInput struct {
	Body UpdatePublicAccessPayload
}

type PublicAccessOutput struct {
	Body UserStatus
}

type UserStatus struct {
	SetupRequired             bool                  `json:"setupRequired" doc:"Whether the app still needs single-user or multi-user setup." example:"false"`
	Mode                      string                `json:"mode,omitempty" doc:"Configured user mode: single or multi." enum:"single,multi" example:"single"`
	RegistrationMode          string                `json:"registrationMode" doc:"Configured registration mode: invite_only or open." enum:"invite_only,open" example:"invite_only"`
	PublicAccess              bool                  `json:"publicAccess" doc:"Whether anonymous read-only access is enabled." example:"false"`
	EmailVerificationRequired bool                  `json:"emailVerificationRequired" doc:"Whether login is blocked until email verification is completed." example:"false"`
	EmailVerificationEmail    string                `json:"emailVerificationEmail,omitempty" doc:"Email address waiting for verification." example:"reader@example.com"`
	User                      *User                 `json:"user,omitempty" doc:"Current user, when a session is active or single-user mode is enabled."`
	MetronPermissions         UserMetronPermissions `json:"metronPermissions" doc:"Current user's Metron endpoint permissions."`
}

type UserStatusOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      UserStatus
}

type UserStatusInput struct {
	Session string `cookie:"comichero_session"`
}

type LogoutUserOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type LogoutUserInput struct {
	Session string `cookie:"comichero_session"`
}

type SetupUsersPayload struct {
	Mode     string `json:"mode" doc:"User mode to enable: single avoids login, multi enables registration and login." enum:"single,multi" example:"multi"`
	Name     string `json:"name,omitempty" doc:"Initial user name for multi-user mode. Existing read status is attached to this user." example:"Justin"`
	Email    string `json:"email,omitempty" doc:"Initial email address for multi-user login." format:"email" example:"reader@example.com"`
	Password string `json:"password,omitempty" doc:"Initial password for multi-user mode." example:"correct horse battery staple"`
}

type SetupUsersInput struct {
	Body SetupUsersPayload
}

type UserCredentialsPayload struct {
	Name                 string `json:"name,omitempty"                 doc:"Display name for registration." example:"Justin"`
	Email                string `json:"email"                          minLength:"1" format:"email" doc:"Email address used to log in." example:"reader@example.com"`
	EmailConfirmation    string `json:"emailConfirmation,omitempty"    format:"email" doc:"Repeated email address required for registration." example:"reader@example.com"`
	Password             string `json:"password"                       minLength:"6" doc:"Password." example:"correct horse battery staple"`
	PasswordConfirmation string `json:"passwordConfirmation,omitempty" minLength:"6" doc:"Repeated password required for registration." example:"correct horse battery staple"`
	InviteToken          string `json:"inviteToken,omitempty"          doc:"Invite token required for registration in multi-user mode."`
}

type RegisterUserInput struct {
	Body UserCredentialsPayload
}

type LoginUserInput struct {
	Body UserCredentialsPayload
}

type VerifyEmailPayload struct {
	Token string `json:"token" minLength:"1" doc:"Email verification token."`
}

type VerifyEmailInput struct {
	Body VerifyEmailPayload
}

type ResendEmailVerificationPayload struct {
	Email    string `json:"email"    minLength:"1" format:"email" doc:"Email address used to log in." example:"reader@example.com"`
	Password string `json:"password" minLength:"6" doc:"Password." example:"correct horse battery staple"`
}

type ResendEmailVerificationInput struct {
	Body ResendEmailVerificationPayload
}

type ForgotPasswordPayload struct {
	Email string `json:"email" minLength:"1" format:"email" doc:"Email address used to log in." example:"reader@example.com"`
}

type ForgotPasswordInput struct {
	Body ForgotPasswordPayload
}

type ResetPasswordPayload struct {
	Token                string `json:"token"                minLength:"1" doc:"Password reset token."`
	Password             string `json:"password"             minLength:"6" doc:"New password." example:"correct horse battery staple"`
	PasswordConfirmation string `json:"passwordConfirmation" minLength:"6" doc:"Repeated new password." example:"correct horse battery staple"`
}

type ResetPasswordInput struct {
	Body ResetPasswordPayload
}

type UpdateAccountPayload struct {
	Name            string `json:"name"                      minLength:"1" doc:"New display name." example:"Justin"`
	CurrentPassword string `json:"currentPassword,omitempty" doc:"Current password, required when changing password in multi-user mode."`
	NewPassword     string `json:"newPassword,omitempty"     doc:"New password. Leave empty to keep the current password."`
}

type UpdateAccountInput struct {
	Body UpdateAccountPayload
}

type DeleteAccountPayload struct {
	CurrentPassword string `json:"currentPassword,omitempty" doc:"Current password for multi-user account deletion."`
}

type DeleteAccountInput struct {
	Body DeleteAccountPayload
}
