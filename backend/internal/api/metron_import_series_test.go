package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestImportMetronSeriesSkipsDetailFetchForExistingComic(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 101)
	`); err != nil {
		t.Fatalf("insert existing comic: %v", err)
	}

	detailRequests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detailRequests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/issue/102/":
			w.Write([]byte(`{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2","cover_date":"2026-02-01"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	output, err := importMetronSeries(ctx, db, client, nil, []metron.Issue{
		{ID: 101, Series: "Series", SeriesYear: 2026, Issue: "1", Publisher: "Publisher"},
		{ID: 102, Series: "Series", SeriesYear: 2026, Issue: "2", Publisher: "Publisher"},
	})
	if err != nil {
		t.Fatalf("importMetronSeries: %v", err)
	}
	if len(output.Body) != 2 {
		t.Fatalf("imported comics = %d; want 2", len(output.Body))
	}
	if detailRequests["/issue/101/"] != 0 {
		t.Fatalf("fetched existing comic detail %d times; want 0", detailRequests["/issue/101/"])
	}
	if detailRequests["/issue/102/"] != 0 {
		t.Fatalf("fetched new comic detail %d times; want 0", detailRequests["/issue/102/"])
	}
}

func TestImportMetronSeriesOptionsControlDetailFetches(t *testing.T) {
	ctx := testUserContext()

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/issue/201/":
			w.Write([]byte(`{"id":201,"series":{"id":401,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1","characters":[{"id":301,"name":"Attached Hero"}]}`))
		case "/series/401/":
			w.Write([]byte(`{"id":401,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"issue_count":1}`))
		case "/character/301/":
			w.Write([]byte(`{"id":301,"name":"Full Hero","desc":"Full profile","alias":["Heroic"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := metron.New(metron.Config{BaseURL: server.URL})
	issues := []metron.Issue{{ID: 201, Series: "Series", SeriesYear: 2026, Issue: "1", Publisher: "Publisher"}}

	quickDB := newMetronImportTestDB(t)
	if _, err := importMetronSeriesWithProgressOptions(ctx, quickDB, client, nil, issues, func(int, int, string) {}, MetronImportOptions{Mode: "quick"}); err != nil {
		t.Fatalf("quick import: %v", err)
	}
	if requests["/issue/201/"] != 0 || requests["/series/401/"] != 0 || requests["/character/301/"] != 0 {
		t.Fatalf("quick requests = %#v; want no issue, series, or character detail calls", requests)
	}

	fullDB := newMetronImportTestDB(t)
	if _, err := importMetronSeriesWithProgressOptions(ctx, fullDB, client, nil, issues, func(int, int, string) {}, MetronImportOptions{Mode: "full"}); err != nil {
		t.Fatalf("full import: %v", err)
	}
	if requests["/issue/201/"] != 1 || requests["/series/401/"] != 1 || requests["/character/301/"] != 1 {
		t.Fatalf("full requests = %#v; want one issue, series, and character detail call", requests)
	}

	var character Character
	if err := fullDB.GetContext(ctx, &character, `SELECT * FROM characters WHERE metron_character_id = 301`); err != nil {
		t.Fatalf("full character: %v", err)
	}
	if character.Name != "Full Hero" || character.Description != "Full profile" {
		t.Fatalf("character = %#v; want full metadata", character)
	}
}

func TestImportLocalSeriesFromMetronUpdatesMetadataAndImportsMissingComics(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO series (id, name, series_year, favorite)
		VALUES (1, 'Series', 2026, 1);
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 101);
	`); err != nil {
		t.Fatalf("insert local series: %v", err)
	}

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.String()]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/series/?name=Series":
			w.Write([]byte(`{"results":[{"id":405,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"volume":2,"year_end":2027,"issue_count":2,"description":"Series metadata"}]}`))
		case "/series/405/":
			w.Write([]byte(`{"id":405,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"volume":2,"year_end":2027,"issue_count":2,"description":"Series metadata"}`))
		case "/series/405/issue_list/":
			w.Write([]byte(`[
				{"id":101,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1"},
				{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2"}
			]`))
		case "/issue/102/":
			w.Write([]byte(`{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := newMetronImportJobStore()
	client := metron.New(metron.Config{BaseURL: server.URL})
	output, err := importLocalSeriesFromMetron(ctx, db, client, nil, store, 1)
	if err != nil {
		t.Fatalf("importLocalSeriesFromMetron: %v", err)
	}

	var current MetronImportJob
	for range 100 {
		var ok bool
		current, ok = store.get(output.Body.ID)
		if !ok {
			t.Fatal("job not found")
		}
		if current.Status == "succeeded" || current.Status == "failed" || current.Status == "canceled" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != "succeeded" {
		t.Fatalf("job status = %q, message = %q; want succeeded", current.Status, current.Message)
	}
	if requests["/issue/101/"] != 0 {
		t.Fatalf("existing issue detail requests = %d; want 0", requests["/issue/101/"])
	}
	if requests["/issue/102/"] != 0 {
		t.Fatalf("new issue detail requests = %d; want 0", requests["/issue/102/"])
	}

	var series ComicSeries
	if err := db.GetContext(ctx, &series, `SELECT * FROM series WHERE id = 1`); err != nil {
		t.Fatalf("get series: %v", err)
	}
	if series.MetronSeriesID == nil || *series.MetronSeriesID != 405 {
		t.Fatalf("metron series id = %#v; want 405", series.MetronSeriesID)
	}
	if series.Description != "Series metadata" || series.IssueCount != 2 || series.Volume != 2 {
		t.Fatalf("series metadata = %#v; want Metron metadata", series)
	}
	if !series.Favorite {
		t.Fatal("series favorite was not preserved")
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 2 {
		t.Fatalf("comic count = %d; want 2", comicCount)
	}
	var linkedComicCount int
	if err := db.GetContext(ctx, &linkedComicCount, `SELECT COUNT(*) FROM comics WHERE series_id = 1`); err != nil {
		t.Fatalf("count linked comics: %v", err)
	}
	if linkedComicCount != 2 {
		t.Fatalf("linked comic count = %d; want 2", linkedComicCount)
	}
}

func TestUpdateSeriesMetronMetadataKeepsLocalRowOnNameYearConflict(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO series (id, name, series_year)
		VALUES (1, 'Local Series', 1976);
		INSERT INTO series (id, name, series_year)
		VALUES (2, 'Metron Series', 1997);
	`); err != nil {
		t.Fatalf("insert series: %v", err)
	}

	err := updateSeriesMetronMetadata(ctx, db, 1, metron.Series{
		ID:          405,
		Name:        "Metron Series",
		YearBegan:   1997,
		Publisher:   "Publisher",
		Volume:      2,
		YearEnd:     2000,
		IssueCount:  259,
		Description: "Metadata",
	})
	if err != nil {
		t.Fatalf("updateSeriesMetronMetadata: %v", err)
	}

	var series ComicSeries
	if err := db.GetContext(ctx, &series, `SELECT * FROM series WHERE id = 1`); err != nil {
		t.Fatalf("get series: %v", err)
	}
	if series.Name != "Local Series" || series.SeriesYear != 1976 {
		t.Fatalf("series identity = %s %d; want local identity preserved", series.Name, series.SeriesYear)
	}
	if series.Publisher != "Publisher" || series.IssueCount != 259 || series.Description != "Metadata" {
		t.Fatalf("series metadata = %#v; want partial metadata applied", series)
	}
}
