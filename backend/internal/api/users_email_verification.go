package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const (
	emailVerificationTokenBytes = 32
	emailVerificationTTL        = 30 * time.Minute
)

type emailVerificationMessage struct {
	To    string
	Link  string
	Token string
}

func requiresEmailVerification(regMode string) bool {
	return regMode == registrationModeOpen
}

func createEmailVerification(ctx context.Context, db sqlx.ExtContext, userID int) (string, string, error) {
	token, err := randomToken(emailVerificationTokenBytes)
	if err != nil {
		return "", "", huma.Error500InternalServerError("failed to create email verification token")
	}
	tokenHash := emailVerificationTokenHash(token)
	expiresAt := time.Now().UTC().Add(emailVerificationTTL).Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
		UPDATE user_email_verifications
		SET used_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND used_at = ''
	`, userID); err != nil {
		return "", "", huma.Error500InternalServerError("failed to reset email verification tokens")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_email_verifications (token_hash, user_id, expires_at)
		VALUES (?, ?, ?)
	`, tokenHash, userID, expiresAt); err != nil {
		return "", "", huma.Error500InternalServerError("failed to save email verification token")
	}
	return token, expiresAt, nil
}

func emailVerificationTokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func verifyEmailToken(ctx context.Context, db *sqlx.DB, token string) (int, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, huma.Error400BadRequest("verification token is required")
	}
	tokenHash := emailVerificationTokenHash(token)
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, huma.Error500InternalServerError("failed to start email verification")
	}
	defer tx.Rollback()

	var row struct {
		UserID    int    `db:"user_id"`
		ExpiresAt string `db:"expires_at"`
	}
	if err := tx.GetContext(ctx, &row, `
		SELECT user_id, expires_at
		FROM user_email_verifications
		WHERE token_hash = ? AND used_at = ''
	`, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return 0, huma.Error401Unauthorized("invalid or expired verification token")
		}
		return 0, huma.Error500InternalServerError("failed to fetch email verification token")
	}
	if row.ExpiresAt <= now {
		_, _ = tx.ExecContext(ctx, `UPDATE user_email_verifications SET used_at = ? WHERE token_hash = ?`, now, tokenHash)
		return 0, huma.Error401Unauthorized("invalid or expired verification token")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET email_verified_at = ?
		WHERE id = ?
	`, now, row.UserID); err != nil {
		return 0, huma.Error500InternalServerError("failed to verify email")
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE user_email_verifications
		SET used_at = ?
		WHERE token_hash = ?
	`, now, tokenHash); err != nil {
		return 0, huma.Error500InternalServerError("failed to consume email verification token")
	}
	if err := tx.Commit(); err != nil {
		return 0, huma.Error500InternalServerError("failed to verify email")
	}
	return row.UserID, nil
}

func sendEmailVerification(ctx context.Context, msg emailVerificationMessage) error {
	_ = ctx
	if strings.TrimSpace(msg.To) == "" {
		return huma.Error500InternalServerError("missing verification email recipient")
	}
	if strings.TrimSpace(os.Getenv("SMTP_HOST")) == "" {
		log.Printf("email verification for %s: %s", msg.To, msg.Link)
		return nil
	}

	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	}
	if from == "" {
		from = "noreply@localhost"
	}

	body := strings.Join([]string{
		fmt.Sprintf("To: %s", msg.To),
		fmt.Sprintf("From: %s", from),
		"Subject: Verify your ComicHero email",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Verify your ComicHero account by opening this link:",
		msg.Link,
		"",
		"Or paste this token into ComicHero:",
		msg.Token,
		"",
		"This token expires in 30 minutes.",
	}, "\r\n")

	addr := host + ":" + port
	var auth smtp.Auth
	username := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	password := os.Getenv("SMTP_PASSWORD")
	if username != "" || password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}
	if err := smtp.SendMail(addr, auth, from, []string{msg.To}, []byte(body)); err != nil {
		log.Printf("failed to send verification email via %s from %s to %s: %v", addr, from, msg.To, err)
		return huma.Error500InternalServerError("failed to send verification email")
	}
	return nil
}

func emailVerificationLink(token string) string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/")
	if base == "" {
		base = "http://localhost:" + strings.TrimSpace(os.Getenv("PORT"))
		if strings.HasSuffix(base, ":") {
			base += "8080"
		}
	}
	link, err := url.Parse(base + "/verify-email")
	if err != nil {
		return base + "/verify-email?token=" + url.QueryEscape(token)
	}
	query := link.Query()
	query.Set("token", token)
	link.RawQuery = query.Encode()
	return link.String()
}
