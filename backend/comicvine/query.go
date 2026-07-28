package comicvine

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Query represents a query builder for Comic Vine API requests
type Query struct {
	filters []Filter
	sorts   []Sort
	limit   int
	offset  int
	fields  []string
	page    int
}

// Filter represents a single filter condition
type Filter struct {
	Field string
	Value string
}

// Sort represents a sort condition
type Sort struct {
	Field     string
	Direction string // "asc" or "desc"
}

// NewQuery creates a new Query builder
func NewQuery() *Query {
	return &Query{
		filters: make([]Filter, 0),
		sorts:   make([]Sort, 0),
		fields:  make([]string, 0),
	}
}

// Filter adds a filter condition
// Example: Filter("name", "Spider-Man")
func (q *Query) Filter(field, value string) *Query {
	q.filters = append(q.filters, Filter{Field: field, Value: value})
	return q
}

// FilterInt adds a filter condition with an integer value
func (q *Query) FilterInt(field string, value int) *Query {
	return q.Filter(field, strconv.Itoa(value))
}

// Sort adds a sort condition
// Example: Sort("date_added", "desc")
func (q *Query) Sort(field, direction string) *Query {
	if direction != "asc" && direction != "desc" {
		direction = "asc"
	}
	q.sorts = append(q.sorts, Sort{Field: field, Direction: direction})
	return q
}

// Limit sets the maximum number of results to return
func (q *Query) Limit(limit int) *Query {
	q.limit = limit
	return q
}

// Offset sets the number of results to skip
func (q *Query) Offset(offset int) *Query {
	q.offset = offset
	return q
}

// Page sets the page number (alternative to offset)
func (q *Query) Page(page int) *Query {
	q.page = page
	return q
}

// Fields specifies which fields to return
func (q *Query) Fields(fields ...string) *Query {
	q.fields = append(q.fields, fields...)
	return q
}

// ToParams converts the query to URL parameters
func (q *Query) ToParams() url.Values {
	params := url.Values{}

	// Add filters
	if len(q.filters) > 0 {
		filterStr := q.buildFilterString()
		params.Set("filter", filterStr)
	}

	// Add sorts
	if len(q.sorts) > 0 {
		sortStr := q.buildSortString()
		params.Set("sort", sortStr)
	}

	// Add limit
	if q.limit > 0 {
		params.Set("limit", strconv.Itoa(q.limit))
	}

	// Add offset
	if q.offset >= 0 && (q.offset > 0 || q.limit > 0) {
		params.Set("offset", strconv.Itoa(q.offset))
	}

	// Add page
	if q.page > 0 {
		params.Set("page", strconv.Itoa(q.page))
	}

	// Add fields
	if len(q.fields) > 0 {
		params.Set("field_list", strings.Join(q.fields, ","))
	}

	return params
}

// buildFilterString builds the filter parameter string
func (q *Query) buildFilterString() string {
	filters := make([]string, len(q.filters))
	for i, f := range q.filters {
		filters[i] = fmt.Sprintf("%s:%s", f.Field, f.Value)
	}
	return strings.Join(filters, ",")
}

// buildSortString builds the sort parameter string
func (q *Query) buildSortString() string {
	sorts := make([]string, len(q.sorts))
	for i, s := range q.sorts {
		sorts[i] = fmt.Sprintf("%s:%s", s.Field, s.Direction)
	}
	return strings.Join(sorts, ",")
}

// Common query helper functions
func (q *Query) FilterByName(name string) *Query {
	return q.Filter("name", name)
}

func (q *Query) FilterByPublisher(publisherID int) *Query {
	return q.FilterInt("publisher", publisherID)
}

func (q *Query) FilterByVolume(volumeID int) *Query {
	return q.FilterInt("volume", volumeID)
}

func (q *Query) SortByDateAdded(direction string) *Query {
	return q.Sort("date_added", direction)
}

func (q *Query) SortByName(direction string) *Query {
	return q.Sort("name", direction)
}

// String returns a string representation of the query
func (q *Query) String() string {
	parts := make([]string, 0)

	if len(q.filters) > 0 {
		parts = append(parts, "filter="+q.buildFilterString())
	}

	if len(q.sorts) > 0 {
		parts = append(parts, "sort="+q.buildSortString())
	}

	if q.limit > 0 {
		parts = append(parts, fmt.Sprintf("limit=%d", q.limit))
	}

	if q.offset > 0 {
		parts = append(parts, fmt.Sprintf("offset=%d", q.offset))
	}

	return strings.Join(parts, "&")
}
