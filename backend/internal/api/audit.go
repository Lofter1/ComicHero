package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"
)

type AuditEvent struct {
	ID         int    `json:"id"         db:"id"`
	UserID     *int   `json:"userId"     db:"user_id"`
	UserName   string `json:"userName"   db:"user_name"`
	Method     string `json:"method"     db:"method"`
	Path       string `json:"path"       db:"path"`
	StatusCode int    `json:"statusCode" db:"status_code"`
	OccurredAt string `json:"occurredAt" db:"occurred_at"`
}

type AuditEventListInput struct {
	UserID int `query:"userId" minimum:"0"`
	Limit  int `query:"limit" minimum:"1" maximum:"500" default:"100"`
}

type AuditEventListOutput struct{ Body []AuditEvent }

// AuditMiddleware records successful state-changing API requests. Reads and
// failed mutations are intentionally excluded so the table represents changes
// accepted by the server rather than general HTTP traffic.
func AuditMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !auditMutationMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)
			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			if status >= http.StatusBadRequest {
				return
			}
			var userID any
			if id, err := currentUserID(r.Context()); err == nil {
				userID = id
			}
			_, _ = db.ExecContext(context.WithoutCancel(r.Context()), `
				INSERT INTO audit_events (user_id, method, path, status_code)
				VALUES (?, ?, ?, ?)
			`, userID, r.Method, r.URL.Path, status)
		})
	}
}

func auditMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func listAuditEvents(ctx context.Context, db *sqlx.DB, input *AuditEventListInput) (*AuditEventListOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT ae.id, ae.user_id, COALESCE(u.name, '') AS user_name, ae.method, ae.path, ae.status_code, ae.occurred_at FROM audit_events ae LEFT JOIN users u ON u.id = ae.user_id`
	args := []any{}
	if input.UserID > 0 {
		query += ` WHERE ae.user_id = ?`
		args = append(args, input.UserID)
	}
	query += ` ORDER BY ae.id DESC LIMIT ?`
	args = append(args, limit)
	events := []AuditEvent{}
	if err := db.SelectContext(ctx, &events, query, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch audit events")
	}
	return &AuditEventListOutput{Body: events}, nil
}
