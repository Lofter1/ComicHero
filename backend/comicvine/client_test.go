package comicvine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	apiKey := "test-api-key"
	client := NewClient(apiKey)

	if client.apiKey != apiKey {
		t.Errorf("Expected apiKey %s, got %s", apiKey, client.apiKey)
	}

	if client.baseURL != defaultBaseURL {
		t.Errorf("Expected baseURL %s, got %s", defaultBaseURL, client.baseURL)
	}

	if client.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", client.maxRetries)
	}

	if client.userAgent != defaultUserAgent {
		t.Errorf("Expected userAgent %s, got %s", defaultUserAgent, client.userAgent)
	}

	// Verify all services are initialized
	if client.Characters() == nil {
		t.Error("Characters service should not be nil")
	}
	if client.Issues() == nil {
		t.Error("Issues service should not be nil")
	}
	if client.Volumes() == nil {
		t.Error("Volumes service should not be nil")
	}
}

func TestClientOptions(t *testing.T) {
	customHTTP := &http.Client{Timeout: 10 * time.Second}
	
	client := NewClient("api-key",
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customHTTP),
		WithRateLimit(5),
		WithRetry(2),
		WithUserAgent("CustomAgent/2.0"),
	)

	if client.baseURL != "https://custom.api.com" {
		t.Errorf("Expected custom base URL, got %s", client.baseURL)
	}

	if client.httpClient != customHTTP {
		t.Error("Expected custom HTTP client")
	}

	if client.maxRetries != 2 {
		t.Errorf("Expected maxRetries 2, got %d", client.maxRetries)
	}

	if client.userAgent != "CustomAgent/2.0" {
		t.Errorf("Expected custom user agent, got %s", client.userAgent)
	}
}

func TestClientGetJSON(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		expectedError  bool
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody: map[string]interface{}{
				"status_code": 1,
				"error":       "OK",
				"results":     []interface{}{},
			},
			expectedError: false,
		},
		{
			name:           "api error response",
			responseStatus: http.StatusOK,
			responseBody: map[string]interface{}{
				"status_code": 100,
				"error":       "Invalid API Key",
			},
			expectedError: true,
		},
		{
			name:           "http error response",
			responseStatus: http.StatusUnauthorized,
			responseBody:   "Unauthorized",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Header.Get("User-Agent") != defaultUserAgent {
					t.Error("Expected User-Agent header")
				}

				apiKey := r.URL.Query().Get("api_key")
				if apiKey == "" {
					t.Error("Expected api_key parameter")
				}

				format := r.URL.Query().Get("format")
				if format != "json" {
					t.Errorf("Expected format json, got %s", format)
				}

				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			
			var result map[string]interface{}
			err := client.getJSON(context.Background(), "/test", nil, &result)

			if tt.expectedError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := client.getJSON(ctx, "/test", nil, nil)
	if err == nil {
		t.Error("Expected context deadline exceeded error")
	}
}

func TestRateLimiting(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status_code": 1,
			"error":       "OK",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", 
		WithBaseURL(server.URL),
		WithRateLimit(10), // 10 requests per second
	)

	start := time.Now()
	for i := 0; i < 10; i++ {
		client.getJSON(context.Background(), "/test", nil, nil)
	}
	elapsed := time.Since(start)

	if elapsed < 900*time.Millisecond {
		t.Errorf("Requests completed too quickly (%v), rate limiting may not be working", elapsed)
	}
}
