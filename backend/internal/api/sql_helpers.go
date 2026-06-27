package api

import (
	"database/sql"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type selectQuery struct {
	base    string
	filters []string
	args    []any
	suffix  string
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

func (q *selectQuery) build() (string, []any) {
	query := q.base
	if len(q.filters) > 0 {
		query += " WHERE " + strings.Join(q.filters, " AND ")
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
