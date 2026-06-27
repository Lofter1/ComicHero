package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"ComicHeroV2-Backend/internal/metron"
)

func newMetronImportTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_reading_list_id INTEGER
		);
		CREATE UNIQUE INDEX idx_reading_orders_metron_reading_list_id
		ON reading_orders(metron_reading_list_id)
		WHERE metron_reading_list_id IS NOT NULL;

		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue INTEGER NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
		CREATE UNIQUE INDEX idx_comics_metron_issue_id
		ON comics(metron_issue_id)
		WHERE metron_issue_id IS NOT NULL;

		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func TestImportMetronComicReusesExistingMetronComic(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)
	issue := metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      1,
		Publisher:  "Publisher",
	}

	first, err := importMetronComic(ctx, db, nil, issue)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := importMetronComic(ctx, db, nil, issue)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.Body.ID != second.Body.ID {
		t.Fatalf("comic ids differ: first=%d second=%d", first.Body.ID, second.Body.ID)
	}

	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if count != 1 {
		t.Fatalf("comic count = %d; want 1", count)
	}
}

func TestImportMetronReadingListReusesExistingOrderAndComics(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)
	list := metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      1,
				Publisher:  "Publisher",
			},
		},
	}

	first, err := importMetronReadingList(ctx, db, nil, list)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := importMetronReadingList(ctx, db, nil, list)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.Body.ID != second.Body.ID {
		t.Fatalf("order ids differ: first=%d second=%d", first.Body.ID, second.Body.ID)
	}

	var orderCount int
	if err := db.GetContext(ctx, &orderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("order count = %d; want 1", orderCount)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}
}

func TestContinueMetronReadingListFillsExistingOrder(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (id, name, description, metron_reading_list_id)
		VALUES (1, 'Event', 'Partial import', 501)
	`); err != nil {
		t.Fatalf("insert partial reading order: %v", err)
	}

	err := continueMetronReadingListWithProgress(ctx, db, nil, metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      1,
				Publisher:  "Publisher",
			},
		},
	}, func(int, int, string) {})
	if err != nil {
		t.Fatalf("continueMetronReadingListWithProgress: %v", err)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}

	var linkedCount int
	if err := db.GetContext(ctx, &linkedCount, `SELECT COUNT(*) FROM reading_order_comics WHERE reading_order_id = 1`); err != nil {
		t.Fatalf("count linked comics: %v", err)
	}
	if linkedCount != 1 {
		t.Fatalf("linked comics = %d; want 1", linkedCount)
	}
}

func TestUpdateComicReadStatus(t *testing.T) {
	ctx := context.Background()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, read)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 0)
	`); err != nil {
		t.Fatalf("insert comic: %v", err)
	}

	detail, err := updateComicReadStatus(ctx, db, 1, true)
	if err != nil {
		t.Fatalf("updateComicReadStatus: %v", err)
	}
	if !detail.Body.Read {
		t.Fatal("comic read status was not updated")
	}
	if detail.Body.Title != "Series (2026) #1" {
		t.Fatalf("comic metadata changed unexpectedly: %#v", detail.Body)
	}
}

func TestImportMetronSeriesSkipsDetailFetchForExistingComic(t *testing.T) {
	ctx := context.Background()
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
		{ID: 101, Series: "Series", SeriesYear: 2026, Issue: 1, Publisher: "Publisher"},
		{ID: 102, Series: "Series", SeriesYear: 2026, Issue: 2, Publisher: "Publisher"},
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
	if detailRequests["/issue/102/"] != 1 {
		t.Fatalf("fetched new comic detail %d times; want 1", detailRequests["/issue/102/"])
	}
}
