package metron

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	client := NewClient()

	if client.baseURL != DefaultBaseURL {
		t.Errorf("expected baseURL %s, got %s", DefaultBaseURL, client.baseURL)
	}
	if client.userAgent != defaultUserAgent {
		t.Errorf("expected userAgent %s, got %s", defaultUserAgent, client.userAgent)
	}
	if client.Arcs() == nil || client.Issues() == nil || client.Series() == nil || client.WishLists() == nil {
		t.Error("expected all resource services to be initialized")
	}
}

func TestClientOptions(t *testing.T) {
	client := NewClient(
		WithBaseURL("https://example.test/api/"),
		WithUserAgent("test-agent/1.0"),
		WithToken("secret-token"),
	)

	if client.baseURL != "https://example.test/api" {
		t.Errorf("expected trailing slash trimmed, got %s", client.baseURL)
	}
	if client.userAgent != "test-agent/1.0" {
		t.Errorf("expected custom user agent, got %s", client.userAgent)
	}
	if client.token != "secret-token" {
		t.Errorf("expected token to be set")
	}
}

func TestAuthorizeTokenTakesPriority(t *testing.T) {
	client := NewClient(
		WithBasicAuth("user", "pass"),
		WithCookie("session-value"),
		WithToken("bearer-token"),
	)

	req, _ := http.NewRequest(http.MethodGet, "https://example.test/", nil)
	client.authorize(req)

	if got := req.Header.Get("Authorization"); got != "Bearer bearer-token" {
		t.Errorf("expected token auth to take priority, got Authorization=%q", got)
	}
}

func TestAuthorizeBasicAuthFallback(t *testing.T) {
	client := NewClient(WithBasicAuth("user", "pass"))

	req, _ := http.NewRequest(http.MethodGet, "https://example.test/", nil)
	client.authorize(req)

	username, password, ok := req.BasicAuth()
	if !ok || username != "user" || password != "pass" {
		t.Errorf("expected basic auth to be applied, got ok=%v user=%q pass=%q", ok, username, password)
	}
}

func TestArcsGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/arc/1/" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("expected bearer auth header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Arc{ID: 1, Name: "Infinity War"})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithToken("test-token"))
	arc, err := client.Arcs().Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arc.Name != "Infinity War" {
		t.Errorf("expected arc name %q, got %q", "Infinity War", arc.Name)
	}
}

func TestArcsAllIssuesWalksPagination(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "", "1":
			_ = json.NewEncoder(w).Encode(PagedResponse[IssueList]{
				Count:   2,
				Next:    "http://" + r.Host + "/arc/1/issue_list/?page=2",
				Results: []IssueList{{ID: 1, Number: "1"}},
			})
		default:
			_ = json.NewEncoder(w).Encode(PagedResponse[IssueList]{
				Count:   2,
				Results: []IssueList{{ID: 2, Number: "2"}},
			})
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	issues, err := client.Arcs().AllIssues(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues across pages, got %d", len(issues))
	}
	if issues[0].ID != 1 || issues[1].ID != 2 {
		t.Errorf("expected issues in page order, got %+v", issues)
	}
	if requests != 2 {
		t.Errorf("expected 2 requests (one per page), got %d", requests)
	}
}

func TestAPIErrorNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.Arcs().Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected an error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("expected IsNotFound() to be true")
	}
}

func TestRateLimitErrorOnTooManyRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Burst-Remaining", "0")
		w.Header().Set("X-RateLimit-Burst-Reset", "1700000000")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.Arcs().Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
}

func TestPriceUnmarshalBothShapes(t *testing.T) {
	var plain Price
	if err := json.Unmarshal([]byte(`"3.99"`), &plain); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain.Amount != "3.99" || plain.Currency != "USD" {
		t.Errorf("expected 3.99/USD, got %+v", plain)
	}

	var object Price
	if err := json.Unmarshal([]byte(`{"amount": 4.99, "currency": "GBP"}`), &object); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if object.Amount != "4.99" || object.Currency != "GBP" {
		t.Errorf("expected 4.99/GBP, got %+v", object)
	}
}

func TestCollectionScrobble(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body ScrobbleRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.IssueID != 42 {
			t.Errorf("expected issue_id 42, got %d", body.IssueID)
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ScrobbleResponse{ID: 7, IsRead: true, Created: true})
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	resp, err := client.Collection().Scrobble(context.Background(), ScrobbleRequest{IssueID: 42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Created || !resp.IsRead {
		t.Errorf("expected created+read response, got %+v", resp)
	}
}
