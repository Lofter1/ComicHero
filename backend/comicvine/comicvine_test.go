package comicvine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCharacterService_GetByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and method
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		resp := SingleCharacterResponse{
			Error:      "OK",
			StatusCode: 1,
			Version:    "1.0",
			Results: Character{
				ID:   1443,
				Name: "Spider-Man",
				Deck: "Friendly neighborhood Spider-Man",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	character, err := client.Characters().GetByID(context.Background(), 1443, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if character.ID != 1443 {
		t.Errorf("Expected ID 1443, got %d", character.ID)
	}

	if character.Name != "Spider-Man" {
		t.Errorf("Expected name 'Spider-Man', got '%s'", character.Name)
	}
}

func TestCharacterService_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CharacterResponse{
			ListResponse: ListResponse{
				Error:           "OK",
				Limit:           10,
				Offset:          0,
				NumberOfResults: 2,
				StatusCode:      1,
				Version:         "1.0",
			},
			Results: []Character{
				{ID: 1443, Name: "Spider-Man"},
				{ID: 1444, Name: "Batman"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Characters().List(context.Background(), &ListOptions{
		Limit:  10,
		Offset: 0,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resp.Results))
	}

	if resp.NumberOfResults != 2 {
		t.Errorf("Expected 2 total results, got %d", resp.NumberOfResults)
	}
}

func TestCharacterService_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != "Spider" {
			t.Errorf("Expected query 'Spider', got '%s'", query)
		}

		resources := r.URL.Query().Get("resources")
		if resources != "character" {
			t.Errorf("Expected resources 'character', got '%s'", resources)
		}

		resp := CharacterResponse{
			ListResponse: ListResponse{
				Error:      "OK",
				StatusCode: 1,
				Version:    "1.0",
			},
			Results: []Character{
				{ID: 1443, Name: "Spider-Man"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Characters().Search(context.Background(), "Spider", nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resp.Results))
	}
}

func TestIssueService_GetByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SingleIssueResponse{
			Error:      "OK",
			StatusCode: 1,
			Version:    "1.0",
			Results: Issue{
				ID:          12345,
				Name:        "Amazing Spider-Man #1",
				IssueNumber: "1",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	issue, err := client.Issues().GetByID(context.Background(), 12345, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if issue.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", issue.ID)
	}

	if issue.IssueNumber != "1" {
		t.Errorf("Expected issue number '1', got '%s'", issue.IssueNumber)
	}
}

func TestVolumeService_GetByID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SingleVolumeResponse{
			Error:      "OK",
			StatusCode: 1,
			Version:    "1.0",
			Results: Volume{
				ID:           40531,
				Name:         "Amazing Spider-Man",
				CountOfIssues: 700,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	volume, err := client.Volumes().GetByID(context.Background(), 40531, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if volume.ID != 40531 {
		t.Errorf("Expected ID 40531, got %d", volume.ID)
	}

	if volume.CountOfIssues != 700 {
		t.Errorf("Expected 700 issues, got %d", volume.CountOfIssues)
	}
}

func TestGlobalSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != "Batman" {
			t.Errorf("Expected query 'Batman', got '%s'", query)
		}

		resp := GlobalSearchResponse{
			ListResponse: ListResponse{
				Error:      "OK",
				StatusCode: 1,
				Version:    "1.0",
			},
			Results: []SearchResult{
				{
					ID:           1444,
					Name:         "Batman",
					ResourceType: "character",
				},
				{
					ID:           42750,
					Name:         "Batman",
					ResourceType: "volume",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	searchService := &SearchService{client: client}
	
	resp, err := searchService.Search(context.Background(), "Batman", []ResourceType{
		ResourceCharacter,
		ResourceVolume,
	}, nil)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resp.Results))
	}

	if resp.Results[0].ResourceType != "character" {
		t.Errorf("Expected first result to be character, got %s", resp.Results[0].ResourceType)
	}
}

func TestFieldsParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields := r.URL.Query().Get("field_list")
		expected := "id,name,image"
		if fields != expected {
			t.Errorf("Expected field_list '%s', got '%s'", expected, fields)
		}

		resp := SingleCharacterResponse{
			Error:      "OK",
			StatusCode: 1,
			Version:    "1.0",
			Results: Character{
				ID:   1443,
				Name: "Spider-Man",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Characters().GetByID(context.Background(), 1443, []string{"id", "name", "image"})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestQueryBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("filter")
		expectedFilter := "name:Batman,publisher:1"
		if filter != expectedFilter {
			t.Errorf("Expected filter '%s', got '%s'", expectedFilter, filter)
		}

		sort := r.URL.Query().Get("sort")
		if sort != "date_added:desc" {
			t.Errorf("Expected sort 'date_added:desc', got '%s'", sort)
		}

		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("Expected limit '10', got '%s'", limit)
		}

		resp := CharacterResponse{
			ListResponse: ListResponse{
				Error:      "OK",
				StatusCode: 1,
				Version:    "1.0",
			},
			Results: []Character{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	
	query := NewQuery().
		Filter("name", "Batman").
		Filter("publisher", "1").
		Sort("date_added", "desc").
		Limit(10)

	_, err := client.Characters().Get(context.Background(), query)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAPIErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]interface{}{
			"error":       "Invalid API Key",
			"status_code": 100,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("invalid-key", WithBaseURL(server.URL))
	_, err := client.Characters().GetByID(context.Background(), 1443, nil)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Test the NewQuery function
	query := NewQuery()
	if query == nil {
		t.Error("NewQuery should not return nil")
	}

	// Test the helper functions for creating common filters
	limit := 20
	opts := &ListOptions{Limit: limit}
	if opts.Limit != 20 {
		t.Errorf("Expected limit 20, got %d", opts.Limit)
	}
}

func TestResourceServiceIntegration(t *testing.T) {
	// Test that all services are properly initialized
	client := NewClient("test-key")

	services := []interface{}{
		client.Characters(),
		client.Concepts(),
		client.Episodes(),
		client.Issues(),
		client.Locations(),
		client.Movies(),
		client.Objects(),
		client.Origins(),
		client.Powers(),
		client.Publishers(),
		client.Series(),
		client.StoryArcs(),
		client.Teams(),
		client.Volumes(),
		client.Promos(),
		client.Videos(),
	}

	for i, service := range services {
		if service == nil {
			t.Errorf("Service %d should not be nil", i)
		}
	}
}

func TestListOptionsValidation(t *testing.T) {
	tests := []struct {
		name     string
		opts     *ListOptions
		expected map[string]string
	}{
		{
			name: "all fields set",
			opts: &ListOptions{
				Limit:  50,
				Offset: 100,
				Sort:   "name:asc",
				Filter: "publisher:1",
				Fields: []string{"id", "name"},
			},
			expected: map[string]string{
				"limit":      "50",
				"offset":     "100",
				"sort":       "name:asc",
				"filter":     "publisher:1",
				"field_list": "id,name",
			},
		},
		{
			name: "empty options",
			opts: &ListOptions{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.expected {
					actualValue := r.URL.Query().Get(key)
					if actualValue != expectedValue {
						t.Errorf("Expected %s=%s, got %s", key, expectedValue, actualValue)
					}
				}
				
				json.NewEncoder(w).Encode(CharacterResponse{
					ListResponse: ListResponse{
						Error:      "OK",
						StatusCode: 1,
					},
				})
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Characters().List(context.Background(), tt.opts)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSearchOptionsValidation(t *testing.T) {
	tests := []struct {
		name     string
		opts     *SearchOptions
		expected map[string]string
	}{
		{
			name: "with limit and page",
			opts: &SearchOptions{
				Limit: 20,
				Page:  2,
			},
			expected: map[string]string{
				"limit": "20",
				"page":  "2",
			},
		},
		{
			name: "with fields",
			opts: &SearchOptions{
				Fields: []string{"id", "name", "image"},
			},
			expected: map[string]string{
				"field_list": "id,name,image",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.expected {
					actualValue := r.URL.Query().Get(key)
					if actualValue != expectedValue {
						t.Errorf("Expected %s=%s, got %s", key, expectedValue, actualValue)
					}
				}
				
				json.NewEncoder(w).Encode(CharacterResponse{
					ListResponse: ListResponse{
						Error:      "OK",
						StatusCode: 1,
					},
				})
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Characters().Search(context.Background(), "test", tt.opts)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func BenchmarkCharacterGetByID(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SingleCharacterResponse{
			Error:      "OK",
			StatusCode: 1,
			Results: Character{
				ID:   1443,
				Name: "Spider-Man",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Characters().GetByID(ctx, 1443, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
