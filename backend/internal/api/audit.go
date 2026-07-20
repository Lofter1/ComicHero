package api

import (
	"context"
	"net/http"
	"strings"

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
	Query     string `query:"q"         doc:"Case-insensitive search across users, methods, paths, and status codes."`
	UserID    int    `query:"userId"    doc:"Filter by user identifier." minimum:"0"`
	System    bool   `query:"system"    doc:"Only return events without an associated user."`
	Method    string `query:"method"    doc:"Filter by HTTP method." enum:"POST,PUT,PATCH,DELETE"`
	Status    string `query:"status"    doc:"Filter by HTTP status family." enum:"1xx,2xx,3xx,4xx,5xx"`
	Sort      string `query:"sort"      doc:"Sort field." enum:"occurredAt,user,action,status"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc"`
	Limit     int    `query:"limit"     doc:"Maximum rows to return, from 1 to 500." minimum:"1" maximum:"500" default:"100"`
	Offset    int    `query:"offset"    doc:"Zero-based row offset for pagination." minimum:"0"`
}

type AuditEventListOutput struct {
	PaginationHeaders
	Body []AuditEvent
}

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
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	query := newSelectQuery(`SELECT ae.id, ae.user_id, COALESCE(u.name, '') AS user_name, ae.method, ae.path, ae.status_code, ae.occurred_at FROM audit_events ae LEFT JOIN users u ON u.id = ae.user_id`)
	if input.Query != "" {
		search := "%" + strings.ToLower(input.Query) + "%"
		query.where(`(
			LOWER(COALESCE(u.name, '')) LIKE ?
			OR LOWER(ae.method) LIKE ?
			OR LOWER(ae.path) LIKE ?
			OR CAST(ae.status_code AS TEXT) LIKE ?
		)`, search, search, search, search)
	}
	if input.UserID > 0 {
		query.where("ae.user_id = ?", input.UserID)
	} else if input.System {
		query.where("ae.user_id IS NULL")
	}
	if input.Method != "" {
		query.where("ae.method = ?", strings.ToUpper(input.Method))
	}
	statusFamilies := map[string]int{"1xx": 100, "2xx": 200, "3xx": 300, "4xx": 400, "5xx": 500}
	if statusMinimum, ok := statusFamilies[input.Status]; ok {
		query.where("ae.status_code >= ? AND ae.status_code < ?", statusMinimum, statusMinimum+100)
	}
	query.orderBy(auditEventListOrder(input.Sort, input.Direction))

	sql, args := query.build()
	total, err := countRows(ctx, db, sql, args)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to count audit events")
	}
	sql += " LIMIT ? OFFSET ?"
	args = append(args, limit+1, offset)
	events := []AuditEvent{}
	if err := db.SelectContext(ctx, &events, sql, args...); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch audit events")
	}
	events, pagination := pageItems(events, limit, offset, total)
	return &AuditEventListOutput{PaginationHeaders: pagination, Body: events}, nil
}

func auditEventListOrder(sort, direction string) string {
	if direction == "" {
		direction = "desc"
	}
	dir := sortDirection(direction)
	switch sort {
	case "user":
		return "ORDER BY LOWER(COALESCE(u.name, '')) " + dir + ", ae.id DESC"
	case "action":
		return "ORDER BY ae.method " + dir + ", ae.path " + dir + ", ae.id DESC"
	case "status":
		return "ORDER BY ae.status_code " + dir + ", ae.id DESC"
	default:
		return "ORDER BY ae.occurred_at " + dir + ", ae.id " + dir
	}
}
