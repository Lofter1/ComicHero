package metron

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUsesBasicAuthAndDocumentedReadingListPaths(t *testing.T) {
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
			w.Write([]byte(`{"results":[{"issue":{"id":11,"series":{"name":"Series"},"number":"1","cover_date":"2026-01-01"}}]}`))
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

	wantPaths := []string{"/reading_list/?name=Event", "/reading_list/7/", "/reading_list/7/items/"}
	if len(paths) != len(wantPaths) {
		t.Fatalf("paths = %#v, want %#v", paths, wantPaths)
	}
	for i, want := range wantPaths {
		if paths[i] != want {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want)
		}
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if _, err := client.SearchSeries(context.Background(), "Series"); err != nil {
		t.Fatalf("SearchSeries: %v", err)
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
	_, err := client.SearchSeries(context.Background(), "Series")
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("err = %T %v, want RateLimitError", err, err)
	}
	if rateLimitErr.RateLimit.BurstRemaining != 0 || rateLimitErr.RateLimit.BurstReset != 1782468300 {
		t.Fatalf("unexpected rate limit error: %#v", rateLimitErr.RateLimit)
	}
}
