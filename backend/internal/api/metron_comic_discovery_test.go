package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestMetronComicDiscoverySettingsRoundTrip(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	settings := MetronComicDiscoverySettings{Enabled: true, Schedule: "monthly", MonthDay: 31, StartTime: "04:30", PublisherName: "Image", SeriesName: "Saga"}
	if err := validateMetronComicDiscoverySettings(&settings); err != nil {
		t.Fatal(err)
	}
	if err := saveMetronComicDiscoverySettings(context.Background(), db, settings); err != nil {
		t.Fatal(err)
	}
	got, err := loadMetronComicDiscoverySettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.MonthDay != 31 || got.PublisherName != "Image" || got.SeriesName != "Saga" {
		t.Fatalf("settings = %+v", got)
	}
}

func TestComicDiscoveryImportsEveryListPageWithoutIssueDetails(t *testing.T) {
	db := newMetronImportTestDB(t)
	requests := map[string]int{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte(`{"count":2,"next":null,"results":[{"id":102,"series":{"id":202,"name":"Saga","year_began":2026,"publisher":{"name":"Image"}},"number":"2","cover_date":"2026-02-01"}]}`))
			return
		}
		if r.URL.Query().Get("publisher_name") != "Image" || r.URL.Query().Get("series_name") != "Saga" || r.URL.Query().Get("modified_gt") == "" {
			t.Errorf("query = %v", r.URL.Query())
		}
		w.Write([]byte(`{"count":2,"next":"` + server.URL + `/issue/?page=2","results":[{"id":101,"series":{"id":202,"name":"Saga","year_began":2026,"publisher":{"name":"Image"}},"number":"1","cover_date":"2026-01-01"}]}`))
	}))
	defer server.Close()
	discovery := NewMetronComicDiscovery(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	discovery.run(context.Background(), MetronComicDiscoverySettings{PublisherName: "Image", SeriesName: "Saga"}, "2026-07-01T00:00:00Z")
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM comics WHERE metron_issue_id IN (101, 102)`); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("imported comics = %d; want 2", count)
	}
	if requests["/issue/"] != 2 {
		t.Fatalf("list requests = %d; want 2", requests["/issue/"])
	}
	if requests["/issue/101/"] != 0 || requests["/issue/102/"] != 0 {
		t.Fatalf("detail requests = %#v", requests)
	}
}

func TestDiscoveryModifiedAfterUsesScheduleWindow(t *testing.T) {
	now := time.Date(2026, time.March, 31, 12, 0, 0, 0, time.UTC)
	if got := discoveryModifiedAfter("daily", now); !got.Equal(now.AddDate(0, 0, -1)) {
		t.Fatalf("daily = %v", got)
	}
	if got := discoveryModifiedAfter("weekly", now); !got.Equal(now.AddDate(0, 0, -7)) {
		t.Fatalf("weekly = %v", got)
	}
	if got, want := discoveryModifiedAfter("monthly", now), time.Date(2026, time.February, 28, 12, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("monthly = %v; want %v", got, want)
	}
}

func TestMonthlyDiscoveryUsesLastDayForShortMonths(t *testing.T) {
	settings := MetronComicDiscoverySettings{Schedule: "monthly", MonthDay: 31}
	februaryEnd := time.Date(2026, time.February, 28, 3, 0, 0, 0, time.UTC)
	if !discoveryScheduleMatches(settings, februaryEnd) {
		t.Fatal("expected February 28 to match day 31 setting")
	}
}
