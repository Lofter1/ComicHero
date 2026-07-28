package comicvine

import (
	"testing"
)

func TestNewQuery(t *testing.T) {
	q := NewQuery()
	if q == nil {
		t.Fatal("NewQuery should not return nil")
	}
	
	if len(q.filters) != 0 {
		t.Error("New query should have no filters")
	}
	
	if len(q.sorts) != 0 {
		t.Error("New query should have no sorts")
	}
}

func TestQueryFilter(t *testing.T) {
	q := NewQuery().Filter("name", "Spider-Man")
	
	if len(q.filters) != 1 {
		t.Fatalf("Expected 1 filter, got %d", len(q.filters))
	}
	
	if q.filters[0].Field != "name" {
		t.Errorf("Expected filter field 'name', got '%s'", q.filters[0].Field)
	}
	
	if q.filters[0].Value != "Spider-Man" {
		t.Errorf("Expected filter value 'Spider-Man', got '%s'", q.filters[0].Value)
	}
}

func TestQueryMultipleFilters(t *testing.T) {
	q := NewQuery().
		Filter("name", "Batman").
		Filter("publisher", "1").
		FilterInt("count_of_issues", 100)
	
	if len(q.filters) != 3 {
		t.Fatalf("Expected 3 filters, got %d", len(q.filters))
	}
	
	expected := []struct {
		field string
		value string
	}{
		{"name", "Batman"},
		{"publisher", "1"},
		{"count_of_issues", "100"},
	}
	
	for i, exp := range expected {
		if q.filters[i].Field != exp.field {
			t.Errorf("Filter %d: expected field '%s', got '%s'", i, exp.field, q.filters[i].Field)
		}
		if q.filters[i].Value != exp.value {
			t.Errorf("Filter %d: expected value '%s', got '%s'", i, exp.value, q.filters[i].Value)
		}
	}
}

func TestQuerySort(t *testing.T) {
	q := NewQuery().Sort("date_added", "desc")
	
	if len(q.sorts) != 1 {
		t.Fatalf("Expected 1 sort, got %d", len(q.sorts))
	}
	
	if q.sorts[0].Field != "date_added" {
		t.Errorf("Expected sort field 'date_added', got '%s'", q.sorts[0].Field)
	}
	
	if q.sorts[0].Direction != "desc" {
		t.Errorf("Expected sort direction 'desc', got '%s'", q.sorts[0].Direction)
	}
}

func TestQuerySortInvalidDirection(t *testing.T) {
	q := NewQuery().Sort("name", "invalid")
	
	if q.sorts[0].Direction != "asc" {
		t.Errorf("Expected default sort direction 'asc', got '%s'", q.sorts[0].Direction)
	}
}

func TestQueryToParams(t *testing.T) {
	tests := []struct {
		name     string
		query    *Query
		expected map[string]string
	}{
		{
			name: "empty query",
			query: NewQuery(),
			expected: map[string]string{},
		},
		{
			name: "with filter",
			query: NewQuery().Filter("name", "Spider-Man"),
			expected: map[string]string{
				"filter": "name:Spider-Man",
			},
		},
		{
			name: "with multiple filters",
			query: NewQuery().
				Filter("name", "Batman").
				Filter("publisher", "1"),
			expected: map[string]string{
				"filter": "name:Batman,publisher:1",
			},
		},
		{
			name: "with sort",
			query: NewQuery().Sort("date_added", "desc"),
			expected: map[string]string{
				"sort": "date_added:desc",
			},
		},
		{
			name: "with limit and offset",
			query: NewQuery().Limit(50).Offset(100),
			expected: map[string]string{
				"limit":  "50",
				"offset": "100",
			},
		},
		{
			name: "with fields",
			query: NewQuery().Fields("id", "name", "image"),
			expected: map[string]string{
				"field_list": "id,name,image",
			},
		},
		{
			name: "complex query",
			query: NewQuery().
				Filter("publisher", "1").
				Filter("name", "Batman").
				Sort("date_added", "desc").
				Limit(20).
				Offset(0).
				Fields("id", "name", "deck"),
			expected: map[string]string{
				"filter":     "publisher:1,name:Batman",
				"sort":       "date_added:desc",
				"limit":      "20",
				"offset":     "0",
				"field_list": "id,name,deck",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.query.ToParams()
			
			for key, expectedValue := range tt.expected {
				actualValue := params.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got %s", key, expectedValue, actualValue)
				}
			}
			
			// Check that no unexpected parameters are present
			if len(params) != len(tt.expected) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expected), len(params))
			}
		})
	}
}

func TestQueryHelperFunctions(t *testing.T) {
	q := NewQuery().
		FilterByName("Spider-Man").
		FilterByPublisher(31).
		SortByDateAdded("desc").
		Limit(10)
	
	params := q.ToParams()
	
	if params.Get("filter") != "name:Spider-Man,publisher:31" {
		t.Errorf("Unexpected filter: %s", params.Get("filter"))
	}
	
	if params.Get("sort") != "date_added:desc" {
		t.Errorf("Unexpected sort: %s", params.Get("sort"))
	}
	
	if params.Get("limit") != "10" {
		t.Errorf("Unexpected limit: %s", params.Get("limit"))
	}
}

func TestQueryString(t *testing.T) {
	q := NewQuery().
		Filter("name", "Batman").
		Sort("date_added", "desc").
		Limit(10)
	
	str := q.String()
	expected := "filter=name:Batman&sort=date_added:desc&limit=10"
	
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}

func TestQueryChaining(t *testing.T) {
	// Test that methods return the same query instance for chaining
	q := NewQuery()
	q2 := q.Filter("name", "test")
	
	if q != q2 {
		t.Error("Filter should return the same query instance for chaining")
	}
}
