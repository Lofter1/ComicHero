package metron

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUsesBasicAuthAndDocumentedListPaths(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass"))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Fatalf("Authorization = %q, want %q", got, wantAuth)
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/reading_list/":
			w.Write([]byte(`{"results":[{"id":7,"name":"Event"}]}`))
		case "/reading_list/7/":
			w.Write([]byte(`{"id":7,"name":"Event","desc":"Big event"}`))
		case "/reading_list/7/items/":
			w.Write([]byte(`{"results":[{"issue_type":"Main Story","issue":{"id":11,"series":{"name":"Series"},"number":"1","cover_date":"2026-01-01"}}]}`))
		case "/arc/":
			w.Write([]byte(`{"results":[{"id":9,"name":"Zero Year"}]}`))
		case "/arc/9/":
			w.Write([]byte(`{"id":9,"name":"Zero Year","desc":"Big arc","image":"https://example.test/arc.jpg"}`))
		case "/arc/9/issue_list/":
			w.Write([]byte(`{"results":[{"id":21,"series":{"name":"Series"},"number":"2","cover_date":"2026-02-01"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		Username: "user",
		Password: "pass",
	})

	lists, err := client.SearchReadingLists(context.Background(), "Event")
	if err != nil {
		t.Fatalf("SearchReadingLists: %v", err)
	}
	if len(lists) != 1 || lists[0].ID != 7 {
		t.Fatalf("unexpected lists: %#v", lists)
	}

	list, err := client.GetReadingList(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetReadingList: %v", err)
	}
	if list.Name != "Event" || len(list.Issues) != 1 || list.Issues[0].ID != 11 {
		t.Fatalf("unexpected reading list: %#v", list)
	}
	if list.Issues[0].Issue != "1" || list.Issues[0].Number != "1" {
		t.Fatalf("issue number = %q/%q, want 1/1", list.Issues[0].Issue, list.Issues[0].Number)
	}
	if len(list.Issues[0].Tags) != 1 || list.Issues[0].Tags[0] != "Main Story" {
		t.Fatalf("issue tags = %#v; want Main Story", list.Issues[0].Tags)
	}

	arcs, err := client.SearchArcs(context.Background(), "Zero")
	if err != nil {
		t.Fatalf("SearchArcs: %v", err)
	}
	if len(arcs) != 1 || arcs[0].ID != 9 {
		t.Fatalf("unexpected arcs: %#v", arcs)
	}

	arc, err := client.GetArc(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetArc: %v", err)
	}
	if arc.Name != "Zero Year" || arc.Image == "" || len(arc.Issues) != 1 || arc.Issues[0].ID != 21 {
		t.Fatalf("unexpected arc: %#v", arc)
	}

	wantPaths := []string{
		"/reading_list/?name=Event",
		"/reading_list/7/",
		"/reading_list/7/items/",
		"/arc/?name=Zero",
		"/arc/9/",
		"/arc/9/issue_list/",
	}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
	for i, want := range wantPaths {
		if paths[i] != want {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want)
		}
	}
}

func TestClientPreservesMetronIssueNumberSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":18030,"series":{"name":"The Amazing Spider-Man","year_began":2018},"number":"50.LR","cover_date":"2020-12-01"}`))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})
	issue, err := client.GetIssue(context.Background(), 18030)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Issue != "50.LR" || issue.Number != "50.LR" {
		t.Fatalf("issue number = %q/%q, want 50.LR/50.LR", issue.Issue, issue.Number)
	}
}

func TestClientUsesConditionalCacheForDetailRequests(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 2 {
			if got := r.Header.Get("If-Modified-Since"); got != "Wed, 12 Feb 2026 10:30:00 GMT" {
				t.Fatalf("If-Modified-Since = %q", got)
			}
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Last-Modified", "Wed, 12 Feb 2026 10:30:00 GMT")
		w.Write([]byte(`{"id":123,"series":{"name":"Series"},"number":"1","cover_date":"2026-01-01"}`))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	first, err := client.GetIssue(context.Background(), 123)
	if err != nil {
		t.Fatalf("first GetIssue: %v", err)
	}
	second, err := client.GetIssue(context.Background(), 123)
	if err != nil {
		t.Fatalf("second GetIssue: %v", err)
	}
	if first.ID != second.ID || second.Series != "Series" {
		t.Fatalf("unexpected cached issue: first=%#v second=%#v", first, second)
	}
}

func TestClientTracksMetronRateLimitHeaders(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Burst-Limit", "10")
		w.Header().Set("X-RateLimit-Burst-Remaining", "4")
		w.Header().Set("X-RateLimit-Burst-Reset", "1782468300")
		w.Header().Set("X-RateLimit-Sustained-Limit", "100")
		w.Header().Set("X-RateLimit-Sustained-Remaining", "75")
		w.Header().Set("X-RateLimit-Sustained-Reset", "1782470000")
		w.Write([]byte(`{"results":[{"id":3,"name":"Series"}]}`))
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})
	if _, err := client.SearchSeries(context.Background(), SeriesSearchOptions{
		Query:     "Series",
		YearBegan: 2018,
		Volume:    2,
	}); err != nil {
		t.Fatalf("SearchSeries: %v", err)
	}
	if want := "/series/?name=Series&volume=2&year_began=2018"; gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}

	rateLimit := client.CurrentRateLimit()
	if rateLimit.BurstLimit != 10 || rateLimit.BurstRemaining != 4 || rateLimit.SustainedRemaining != 75 {
		t.Fatalf("unexpected rate limit: %#v", rateLimit)
	}
}

func TestClientReturnsRateLimitErrorOnTooManyRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Burst-Limit", "10")
		w.Header().Set("X-RateLimit-Burst-Remaining", "0")
		w.Header().Set("X-RateLimit-Burst-Reset", "1782468300")
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})
	_, err := client.SearchSeries(context.Background(), SeriesSearchOptions{Query: "Series"})
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("err = %T %v, want RateLimitError", err, err)
	}
	if rateLimitErr.RateLimit.BurstRemaining != 0 || rateLimitErr.RateLimit.BurstReset != 1782468300 {
		t.Fatalf("unexpected rate limit error: %#v", rateLimitErr.RateLimit)
	}
}

func TestSearchModifiedIssuesUsesFiltersAndAllPages(t *testing.T) {
	var paths []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte(`{"count":2,"next":null,"results":[{"id":2,"series":{"name":"Saga"},"number":"2"}]}`))
			return
		}
		w.Write([]byte(`{"count":2,"next":"` + server.URL + `/issue/?page=2","results":[{"id":1,"series":{"name":"Saga"},"number":"1"}]}`))
	}))
	defer server.Close()
	client := New(Config{BaseURL: server.URL})
	issues, err := client.SearchModifiedIssues(context.Background(), IssueModifiedSearchOptions{ModifiedAfter: "2026-07-01T12:00:00Z", PublisherName: "Image", SeriesName: "Saga"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("issues = %d; want 2", len(issues))
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %#v", paths)
	}
	if paths[0] != "/issue/?modified_gt=2026-07-01T12%3A00%3A00Z&publisher_name=Image&series_name=Saga" {
		t.Fatalf("first path = %q", paths[0])
	}
	if paths[1] != "/issue/?page=2" {
		t.Fatalf("second path = %q", paths[1])
	}
}
