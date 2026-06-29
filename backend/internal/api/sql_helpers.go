package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

type PaginationHeaders struct {
	PageLimit  string `header:"X-Page-Limit"  doc:"Number of rows requested for this page."`
	PageOffset string `header:"X-Page-Offset" doc:"Zero-based row offset for this page."`
	HasMore    string `header:"X-Has-More"    doc:"Whether another page is available."`
	TotalCount string `header:"X-Total-Count" doc:"Total matching rows before pagination."`
}

type selectQuery struct {
	base    string
	filters []string
	args    []any
	group   string
	suffix  string
}

func paginatedQuery(query string, args []any, limit, offset int) (string, []any, int, int) {
	limit, offset = normalizePagination(limit, offset)
	return query + " LIMIT ? OFFSET ?", append(args, limit+1, offset), limit, offset
}

func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func countRows(ctx context.Context, db *sqlx.DB, query string, args []any) (int, error) {
	var total int
	if err := db.GetContext(ctx, &total, "SELECT COUNT(*) FROM ("+query+") paged_rows", args...); err != nil {
		return 0, err
	}
	return total, nil
}

func pageItems[T any](items []T, limit, offset, total int) ([]T, PaginationHeaders) {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return items, PaginationHeaders{
		PageLimit:  strconv.Itoa(limit),
		PageOffset: strconv.Itoa(offset),
		HasMore:    strconv.FormatBool(hasMore),
		TotalCount: strconv.Itoa(total),
	}
}

func newSelectQuery(base string) *selectQuery {
	return &selectQuery{base: base}
}

func (q *selectQuery) where(condition string, args ...any) {
	q.filters = append(q.filters, condition)
	q.args = append(q.args, args...)
}

func (q *selectQuery) orderBy(order string) {
	q.suffix = order
}

func (q *selectQuery) groupBy(group string) {
	q.group = group
}

func (q *selectQuery) build() (string, []any) {
	query := q.base
	if len(q.filters) > 0 {
		query += " WHERE " + strings.Join(q.filters, " AND ")
	}
	if q.group != "" {
		query += " " + q.group
	}
	if q.suffix != "" {
		query += " " + q.suffix
	}
	return query, q.args
}

func parseOptionalBool(value, field string) (bool, bool, error) {
	if value == "" {
		return false, false, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false, huma.Error400BadRequest(field + " must be true or false")
	}
	return parsed, true, nil
}

func requireRowsAffected(result sql.Result, notFoundMessage string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return huma.Error500InternalServerError("failed to check affected rows")
	}
	if rows == 0 {
		return huma.Error404NotFound(notFoundMessage)
	}
	return nil
}
