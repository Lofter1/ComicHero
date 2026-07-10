package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestParseOptionalBool(t *testing.T) {
	value, ok, err := parseOptionalBool("", "favorite")
	if err != nil || ok || value {
		t.Fatalf("empty value = %v, %v, %v; want false, false, nil", value, ok, err)
	}

	value, ok, err = parseOptionalBool("true", "favorite")
	if err != nil || !ok || !value {
		t.Fatalf("true value = %v, %v, %v; want true, true, nil", value, ok, err)
	}

	if _, _, err := parseOptionalBool("sometimes", "favorite"); err == nil {
		t.Fatal("invalid bool returned nil error")
	}
}

func TestPaginationHelpers(t *testing.T) {
	query, args, limit, offset := paginatedQuery("SELECT * FROM comics", []any{"arg"}, 250, -12)
	if query != "SELECT * FROM comics LIMIT ? OFFSET ?" {
		t.Fatalf("query = %q", query)
	}
	if limit != maxPageLimit || offset != 0 {
		t.Fatalf("limit/offset = %d/%d; want %d/0", limit, offset, maxPageLimit)
	}
	if len(args) != 3 || args[1] != maxPageLimit+1 || args[2] != 0 {
		t.Fatalf("args = %#v; want original arg plus page limit+1 and offset", args)
	}

	items, headers := pageItems([]int{1, 2, 3}, 2, 10, 42)
	if len(items) != 2 || items[0] != 1 || items[1] != 2 {
		t.Fatalf("items = %#v; want first two", items)
	}
	if headers.PageLimit != "2" || headers.PageOffset != "10" || headers.HasMore != "true" || headers.TotalCount != "42" {
		t.Fatalf("headers = %#v; want limit 2, offset 10, has more true, total 42", headers)
	}
}

func testUserContext() context.Context {
	return context.WithValue(context.Background(), contextUserIDKey{}, 1)
}

func TestComicListQuery(t *testing.T) {
	query, args, err := comicListQuery(&ComicListInput{
		Query:          "bat",
		Series:         "Detective",
		Publisher:      "DC",
		Read:           "false",
		ReadingOrderID: 12,
	}, 1)
	if err != nil {
		t.Fatalf("comicListQuery returned error: %v", err)
	}

	for _, fragment := range []string{
		"c.series LIKE ?",
		"c.series_year AS TEXT",
		"c.issue AS TEXT",
		"c.publisher LIKE ?",
		"COALESCE(uc.read, 0) = ?",
		"roc.reading_order_id = ?",
		"ORDER BY c.series, c.series_year, CAST(c.issue AS REAL), c.issue",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q: %s", fragment, query)
		}
	}
	if len(args) != 11 {
		t.Fatalf("len(args) = %d; want 11", len(args))
	}

	query, _, err = comicListQuery(&ComicListInput{Status: "read,skipped"}, 1)
	if err != nil {
		t.Fatalf("comicListQuery status returned error: %v", err)
	}
	for _, fragment := range []string{
		"COALESCE(uc.read, 0) = 1",
		"COALESCE(uc.skipped, 0) = 1",
		" OR ",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("status query missing %q: %s", fragment, query)
		}
	}
}

func TestReadingOrderHelpers(t *testing.T) {
	progress := computeProgress([]ReadingOrderComic{
		{Comic: Comic{Read: true}},
		{Comic: Comic{Read: false}},
	})
	if progress != 0.5 {
		t.Fatalf("progress = %v; want 0.5", progress)
	}

	input := &SetReadingOrderComicsInput{}
	input.Body.ComicIDs = []int{1, 2}

	items := readingOrderComicItems(input)
	if len(items) != 2 || items[0].ComicID != 1 || items[1].ComicID != 2 {
		t.Fatalf("items = %#v; want comic IDs 1, 2", items)
	}

	ids := readingOrderComicIDs([]ReadingOrderComicPayload{
		{ComicID: 3},
		{ComicID: 3},
	})
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 3 {
		t.Fatalf("ids = %#v; want duplicate IDs preserved", ids)
	}

	if order := readingOrderListOrder("rating", "desc"); order != "ORDER BY rating DESC, ro.name DESC" {
		t.Fatalf("rating order = %q", order)
	}
}

func TestUserStatisticsAndAchievements(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			metron_issue_id INTEGER
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_arc_starts (
			arc_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (arc_id, user_id)
		);
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE user_series_starts (
			series_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (series_id, user_id)
		);
		CREATE TABLE characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_character_starts (
			character_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (character_id, user_id)
		);
		CREATE TABLE comic_characters (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			PRIMARY KEY (comic_id, character_id)
		);
		INSERT INTO users (id, name, is_default) VALUES (1, 'Reader', 1), (2, 'Other', 0);
		INSERT INTO comics (id, series, series_year, issue, publisher)
		VALUES
			(1, 'Alpha', 2020, '1', 'Pub A'),
			(2, 'Alpha', 2020, '2', 'Pub A'),
			(3, 'Beta', 2021, '1', 'Pub B'),
			(4, 'Gamma', 2022, '1', 'Pub C');
		INSERT INTO user_comics (comic_id, user_id, read, read_at)
		VALUES
			(1, 1, 1, '2026-07-01T10:00:00Z'),
			(2, 1, 1, '2026-07-02T12:30:00Z'),
			(4, 2, 1, '2026-07-03T08:00:00Z');
		INSERT INTO reading_orders (id, name, author_user_id) VALUES (1, 'Alpha order', 1), (2, 'Other order', 2);
		INSERT INTO user_reading_orders (reading_order_id, user_id) VALUES (1, 1);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position)
		VALUES (1, 1, 1), (1, 2, 2), (2, 3, 1);
		INSERT INTO arcs (id, name) VALUES (1, 'Alpha arc'), (2, 'Beta arc');
		INSERT INTO user_arc_starts (arc_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:30:00Z');
		INSERT INTO arc_comics (arc_id, comic_id, position)
		VALUES (1, 1, 1), (1, 2, 2), (2, 3, 1);
		INSERT INTO user_series_starts (series_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:45:00Z');
		INSERT INTO characters (id, name) VALUES (1, 'Hero'), (2, 'Sidekick'), (3, 'Cameo');
		INSERT INTO user_character_starts (character_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:50:00Z');
		INSERT INTO comic_characters (comic_id, character_id)
		VALUES (1, 1), (1, 2), (2, 2), (3, 3);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	result, err := getAccountStatistics(ctx, db)
	if err != nil {
		t.Fatalf("getAccountStatistics: %v", err)
	}
	stats := result.Body.Statistics
	if stats.TotalComics != 4 || stats.ReadComics != 2 || stats.UnreadComics != 2 {
		t.Fatalf("comic counts = total %d read %d unread %d; want 4, 2, 2", stats.TotalComics, stats.ReadComics, stats.UnreadComics)
	}
	if stats.ReadProgress != 0.5 {
		t.Fatalf("read progress = %v; want 0.5", stats.ReadProgress)
	}
	if stats.FirstReadAt != "2026-07-01T10:00:00Z" || stats.LastReadAt != "2026-07-02T12:30:00Z" {
		t.Fatalf("read timestamps = %q/%q; want first and latest reader timestamps", stats.FirstReadAt, stats.LastReadAt)
	}
	if stats.DistinctReadSeries != 1 || stats.DistinctReadPublishers != 1 {
		t.Fatalf("distinct read series/publishers = %d/%d; want 1/1", stats.DistinctReadSeries, stats.DistinctReadPublishers)
	}
	if stats.CompletedSeries != 1 {
		t.Fatalf("completed series = %d; want 1", stats.CompletedSeries)
	}
	if stats.AuthoredReadingOrders != 1 || stats.StartedReadingOrders != 1 || stats.CompletedReadingOrders != 1 {
		t.Fatalf("reading order stats = authored %d started %d completed %d; want 1/1/1", stats.AuthoredReadingOrders, stats.StartedReadingOrders, stats.CompletedReadingOrders)
	}
	if stats.StartedArcs != 1 || stats.CompletedArcs != 1 || stats.StartedSeries != 1 || stats.StartedCharacters != 1 {
		t.Fatalf("started/completed stats = arcs %d/%d series %d characters %d; want 1/1/1/1", stats.StartedArcs, stats.CompletedArcs, stats.StartedSeries, stats.StartedCharacters)
	}
	if stats.CharactersMet != 2 {
		t.Fatalf("characters met = %d; want 2", stats.CharactersMet)
	}

	achievements := map[string]Achievement{}
	for _, achievement := range result.Body.Achievements {
		achievements[achievement.ID] = achievement
	}
	for _, id := range []string{"first-read", "reading-order-finisher", "arc-explorer", "series-finisher", "curator"} {
		if !achievements[id].Earned {
			t.Fatalf("achievement %q not earned", id)
		}
	}
	for _, id := range []string{"reading-order-starter", "arc-starter", "series-starter", "character-starter"} {
		if !achievements[id].Earned {
			t.Fatalf("started achievement %q not earned", id)
		}
		if achievements[id].EarnedAt == "" {
			t.Fatalf("started achievement %q missing earned timestamp", id)
		}
	}
	if achievements["first-read"].EarnedAt != "2026-07-01T10:00:00Z" {
		t.Fatalf("first-read earned at = %q; want first read timestamp", achievements["first-read"].EarnedAt)
	}
	if achievements["page-turner"].Earned {
		t.Fatalf("page-turner earned with %d reads; want locked", achievements["page-turner"].Progress)
	}
	if achievements["page-turner"].EarnedAt != "" {
		t.Fatalf("page-turner earned at = %q; want empty while locked", achievements["page-turner"].EarnedAt)
	}
}

func TestDashboardNextComicAdvancesAfterRead(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL DEFAULT '',
			email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
			password_hash TEXT NOT NULL DEFAULT '',
			is_default INTEGER NOT NULL DEFAULT 0,
			is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_issue_id INTEGER,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			rating REAL NOT NULL,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL,
			comic_id INTEGER NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL,
			child_reading_order_id INTEGER NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_arc_starts (
			arc_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (arc_id, user_id)
		);
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL,
			comic_id INTEGER NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_character_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_character_starts (
			character_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (character_id, user_id)
		);
		CREATE TABLE comic_characters (
			comic_id INTEGER NOT NULL,
			character_id INTEGER NOT NULL,
			PRIMARY KEY (comic_id, character_id)
		);
		CREATE TABLE user_series_starts (
			series_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (series_id, user_id)
		);

		INSERT INTO users (id, name, is_default) VALUES (1, 'Reader', 1);
		INSERT INTO series (id, name, series_year) VALUES (1, 'Alpha', 2020);
		INSERT INTO comics (id, series_id, series, series_year, issue, publisher)
		VALUES (1, 1, 'Alpha', 2020, '1', 'Pub'), (2, 1, 'Alpha', 2020, '2', 'Pub');
		INSERT INTO reading_orders (id, name, author_user_id) VALUES (1, 'Alpha order', 1);
		INSERT INTO user_reading_orders (reading_order_id, user_id, started_at)
		VALUES (1, 1, '2026-07-01T10:00:00Z');
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position)
		VALUES (1, 1, 1), (1, 2, 2);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	dashboard, err := getDashboard(ctx, db)
	if err != nil {
		t.Fatalf("getDashboard: %v", err)
	}
	if len(dashboard.Body.Items) != 1 {
		t.Fatalf("dashboard items = %d; want 1", len(dashboard.Body.Items))
	}
	if got := dashboard.Body.Items[0].NextComic.ID; got != 1 {
		t.Fatalf("next comic = %d; want 1", got)
	}

	read := true
	input := &UpdateComicReadInput{}
	input.Body.Read = &read
	if _, err := updateComicReadStatus(ctx, db, 1, input); err != nil {
		t.Fatalf("updateComicReadStatus: %v", err)
	}

	dashboard, err = getDashboard(ctx, db)
	if err != nil {
		t.Fatalf("getDashboard after read: %v", err)
	}
	if got := dashboard.Body.Items[0].NextComic.ID; got != 2 {
		t.Fatalf("next comic after read = %d; want 2", got)
	}
}

func TestDashboardNextComicUsesAscendingCoverDate(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT NOT NULL, series_year INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY, metron_issue_id INTEGER, series_id INTEGER,
			series TEXT NOT NULL, series_year INTEGER NOT NULL DEFAULT 0, issue TEXT NOT NULL,
			publisher TEXT NOT NULL DEFAULT '', cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL, user_id INTEGER NOT NULL, read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (comic_id, user_id)
		);
		CREATE TABLE arcs (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE arc_comics (arc_id INTEGER NOT NULL, comic_id INTEGER NOT NULL, position INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE user_arc_starts (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL);
		CREATE TABLE characters (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE comic_characters (comic_id INTEGER NOT NULL, character_id INTEGER NOT NULL);
		CREATE TABLE user_character_starts (character_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL);
		CREATE TABLE user_series_starts (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL);

		INSERT INTO series (id, name, series_year) VALUES (1, 'Release order', 2020);
		INSERT INTO comics (id, series_id, series, series_year, issue, cover_date) VALUES
			(1, 1, 'Release order', 2020, '1', '2026-03-01'),
			(2, 1, 'Release order', 2020, '2', '2026-01-01'),
			(3, 1, 'Release order', 2020, '3', '2026-02-01');
		INSERT INTO arcs (id, name) VALUES (1, 'Arc');
		INSERT INTO arc_comics (arc_id, comic_id, position) VALUES (1, 1, 1), (1, 2, 3), (1, 3, 2);
		INSERT INTO user_arc_starts (arc_id, user_id, started_at) VALUES (1, 1, '2026-01-01');
		INSERT INTO characters (id, name) VALUES (1, 'Hero');
		INSERT INTO comic_characters (comic_id, character_id) VALUES (1, 1), (2, 1), (3, 1);
		INSERT INTO user_character_starts (character_id, user_id, started_at) VALUES (1, 1, '2026-01-01');
		INSERT INTO user_series_starts (series_id, user_id, started_at) VALUES (1, 1, '2026-01-01');
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	loaders := []struct {
		name string
		load func(context.Context, *sqlx.DB, int) ([]DashboardItem, error)
	}{
		{name: "arc", load: dashboardArcs},
		{name: "character", load: dashboardCharacters},
		{name: "series", load: dashboardSeries},
	}
	for _, loader := range loaders {
		t.Run(loader.name, func(t *testing.T) {
			items, err := loader.load(ctx, db, 1)
			if err != nil {
				t.Fatalf("load dashboard items: %v", err)
			}
			if len(items) != 1 || items[0].NextComic == nil {
				t.Fatalf("items = %#v; want one item with a next comic", items)
			}
			if got := items[0].NextComic.ID; got != 2 {
				t.Fatalf("next comic = %d; want earliest release-date comic 2", got)
			}
		})
	}
}

func TestReadingOrderEntriesCanNestOrdersBetweenComics(t *testing.T) {
	ctx := testUserContext()
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
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE user_arc_starts (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (arc_id, user_id));
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (id, name, is_default) VALUES (1, 'Default', 1);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
			CHECK (parent_reading_order_id <> child_reading_order_id)
		);
		INSERT INTO comics (series, series_year, issue, publisher)
		VALUES ('Parent', 2026, '1', 'Publisher'),
			('Child', 2026, '2', 'Publisher'),
			('Parent', 2026, '3', 'Publisher');
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	parent, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Child"})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	childInput := &SetReadingOrderComicsInput{ID: child.Body.ID}
	childInput.Body.Entries = []ReadingOrderEntryPayload{{Type: "comic", ComicID: 2}}
	if _, err := setReadingOrderComics(ctx, db, childInput); err != nil {
		t.Fatalf("set child entries: %v", err)
	}

	parentInput := &SetReadingOrderComicsInput{ID: parent.Body.ID}
	parentInput.Body.Entries = []ReadingOrderEntryPayload{
		{Type: "comic", ComicID: 1},
		{Type: "readingOrder", ReadingOrderID: child.Body.ID, Comment: "Crossover break"},
		{Type: "comic", ComicID: 3},
	}
	detail, err := setReadingOrderComics(ctx, db, parentInput)
	if err != nil {
		t.Fatalf("set parent entries: %v", err)
	}

	if len(detail.Body.Entries) != 3 || detail.Body.Entries[1].Type != "readingOrder" {
		t.Fatalf("entries = %#v; want nested reading order in the middle", detail.Body.Entries)
	}
	if detail.Body.Entries[1].Comment != "Crossover break" {
		t.Fatalf("nested order note = %q; want Crossover break", detail.Body.Entries[1].Comment)
	}
	if len(detail.Body.Comics) != 3 {
		t.Fatalf("expanded comics = %d; want 3", len(detail.Body.Comics))
	}
	for i, issue := range []string{"1", "2", "3"} {
		if detail.Body.Comics[i].Issue != issue {
			t.Fatalf("comic %d issue = %q; want %q", i, detail.Body.Comics[i].Issue, issue)
		}
	}
	if detail.Body.Comics[1].Comment != "From Child: Crossover break" {
		t.Fatalf("nested comic comment = %q; want nested order note", detail.Body.Comics[1].Comment)
	}
}

func TestCopyReadingOrderCreatesCurrentUserOwnedCopy(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, name, is_default) VALUES (2, 'Other', 0);
		INSERT INTO comics (id, series, series_year, issue, publisher)
		VALUES
			(1, 'Source', 2026, '1', 'Publisher'),
			(2, 'Source', 2026, '2', 'Publisher');
		INSERT INTO reading_orders (id, name, description, image, favorite, author_user_id)
		VALUES
			(10, 'Other list', 'Read this one', '/covers/list.jpg', 1, 2),
			(11, 'Nested list', 'Nested notes', '', 0, 2);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note, tags)
		VALUES
			(10, 1, 1, 'Start here', 'Main'),
			(11, 2, 1, 'Nested comic', 'Tie-in');
		INSERT INTO reading_order_children (parent_reading_order_id, child_reading_order_id, position, note)
		VALUES (10, 11, 2, 'Then this');
	`); err != nil {
		t.Fatalf("seed copy source: %v", err)
	}

	copied, err := copyReadingOrder(ctx, db, 10)
	if err != nil {
		t.Fatalf("copyReadingOrder: %v", err)
	}

	if copied.Body.ID == 10 {
		t.Fatal("copied reading order reused source id")
	}
	if copied.Body.AuthorUserID == nil || *copied.Body.AuthorUserID != 1 {
		t.Fatalf("copied author = %#v; want current user 1", copied.Body.AuthorUserID)
	}
	if copied.Body.Name != "Other list (Copy)" {
		t.Fatalf("copied name = %q; want copy suffix", copied.Body.Name)
	}
	if copied.Body.Description != "Read this one" || copied.Body.Image != "/covers/list.jpg" {
		t.Fatalf("copied metadata = description %q image %q", copied.Body.Description, copied.Body.Image)
	}
	if copied.Body.Favorite {
		t.Fatal("copied favorite = true; want false for current user's new copy")
	}
	if !copied.Body.CanEdit {
		t.Fatal("copied canEdit = false; want current user to edit their copy")
	}
	if len(copied.Body.Entries) != 2 {
		t.Fatalf("copied entries = %d; want 2", len(copied.Body.Entries))
	}
	if copied.Body.Entries[0].Comic == nil || copied.Body.Entries[0].Comic.ID != 1 {
		t.Fatalf("first copied entry = %#v; want comic 1", copied.Body.Entries[0])
	}
	if copied.Body.Entries[0].Comic.Comment != "Start here" || copied.Body.Entries[0].Comic.Tags != "Main" {
		t.Fatalf("first copied comic note/tags = %q/%q", copied.Body.Entries[0].Comic.Comment, copied.Body.Entries[0].Comic.Tags)
	}
	if copied.Body.Entries[1].ReadingOrder == nil || copied.Body.Entries[1].ReadingOrder.ID != 11 {
		t.Fatalf("second copied entry = %#v; want nested order 11", copied.Body.Entries[1])
	}
	if copied.Body.Entries[1].Comment != "Then this" {
		t.Fatalf("copied nested note = %q; want Then this", copied.Body.Entries[1].Comment)
	}
}

func TestReadingOrderWritesRequireAuthor(t *testing.T) {
	db := setupReadingOrderCBLTestDB(t)
	ctx := testUserContext()
	ownerCtx := context.WithValue(ctx, contextUserIDKey{}, 2)
	otherCtx := context.WithValue(ctx, contextUserIDKey{}, 1)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, name, is_default) VALUES (2, 'Owner', 0)`); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	created, err := createReadingOrder(ownerCtx, db, nil, ReadingOrderPayload{Name: "Owner list"})
	if err != nil {
		t.Fatalf("createReadingOrder: %v", err)
	}
	if created.Body.AuthorUserID == nil || *created.Body.AuthorUserID != 2 {
		t.Fatalf("author user id = %#v; want 2", created.Body.AuthorUserID)
	}

	view, err := getReadingOrder(otherCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("getReadingOrder as non-author: %v", err)
	}
	if view.Body.CanEdit {
		t.Fatalf("non-author canEdit = true; want false")
	}

	if _, err := updateReadingOrder(otherCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Nope"}); err == nil {
		t.Fatalf("updateReadingOrder as non-author succeeded; want error")
	}
	if _, err := db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = 1`); err != nil {
		t.Fatalf("promote admin: %v", err)
	}
	adminView, err := getReadingOrder(otherCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("getReadingOrder as admin: %v", err)
	}
	if !adminView.Body.CanEdit {
		t.Fatalf("admin canEdit = false; want true")
	}
	if _, err := updateReadingOrder(otherCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Admin update"}); err != nil {
		t.Fatalf("updateReadingOrder as admin: %v", err)
	}
	if _, err := updateReadingOrder(ownerCtx, db, nil, created.Body.ID, ReadingOrderPayload{Name: "Updated"}); err != nil {
		t.Fatalf("updateReadingOrder as author: %v", err)
	}
}

func TestReadingOrderCoverUploadResizesAndDeletesUnusedOldCover(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	covers := NewCoverCache(t.TempDir(), "/covers")

	created, err := createReadingOrder(ctx, db, covers, ReadingOrderPayload{
		Name:           "Cover list",
		CoverImageData: imageDataURL("image/png", testPNG(t, 1200, 1800)),
	})
	if err != nil {
		t.Fatalf("createReadingOrder: %v", err)
	}
	oldImage := created.Body.Image
	if oldImage == "" {
		t.Fatal("created reading order missing cover image")
	}
	oldPath := coverPath(t, covers, oldImage)
	assertCoverMaxDimension(t, oldPath, coverMaxDimension)

	updated, err := updateReadingOrder(ctx, db, covers, created.Body.ID, ReadingOrderPayload{
		Name:           "Cover list",
		CoverImageData: imageDataURL("image/jpeg", testJPEG(t, 300, 450)),
	})
	if err != nil {
		t.Fatalf("updateReadingOrder: %v", err)
	}
	if updated.Body.Image == "" {
		t.Fatal("updated reading order missing cover image")
	}
	if updated.Body.Image == oldImage {
		t.Fatalf("image was not replaced: %q", updated.Body.Image)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old cover still exists or stat failed: %v", err)
	}
	assertCoverMaxDimension(t, coverPath(t, covers, updated.Body.Image), 450)
}

func TestReadingOrderCBLImportMatchesLocalComicsAndReportsUnmatched(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)

	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date)
		VALUES ('Frank Miller''s RoboCop', 2003, '1', 'Avatar Press', '2003-07-01'),
			('Frank Miller''s RoboCop', 2003, '2', 'Avatar Press', '2003-08-01');
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	input := &ReadingOrderCBLImportInput{}
	input.Body.Filename = "fallback.cbl"
	input.Body.Content = `<?xml version="1.0" encoding="utf-8"?>
<ReadingList>
	<Name>RoboCop Publication Order</Name>
	<NumIssues>3</NumIssues>
	<Books>
		<Book Series="Frank Miller&apos;s RoboCop" Number="1" Volume="2003" Year="2003" />
		<Book Series="Frank Miller&apos;s RoboCop" Number="2" Volume="2003" Year="2003" />
		<Book Series="Missing Series" Number="1" Volume="2003" Year="2003" />
	</Books>
	<Matchers />
</ReadingList>`

	result, err := importReadingOrderCBL(ctx, db, input)
	if err != nil {
		t.Fatalf("importReadingOrderCBL: %v", err)
	}

	if result.Body.ReadingOrder.Name != "RoboCop Publication Order" {
		t.Fatalf("name = %q; want CBL name", result.Body.ReadingOrder.Name)
	}
	if result.Body.MatchedCount != 2 || result.Body.UnmatchedCount != 1 {
		t.Fatalf("matched/unmatched = %d/%d; want 2/1", result.Body.MatchedCount, result.Body.UnmatchedCount)
	}
	if len(result.Body.ReadingOrder.Comics) != 2 {
		t.Fatalf("imported comics = %d; want 2", len(result.Body.ReadingOrder.Comics))
	}
	if result.Body.ReadingOrder.Comics[0].Issue != "1" || result.Body.ReadingOrder.Comics[1].Issue != "2" {
		t.Fatalf("imported issue order = %#v; want 1 then 2", result.Body.ReadingOrder.Comics)
	}
	if len(result.Body.Unmatched) != 1 || result.Body.Unmatched[0].Series != "Missing Series" {
		t.Fatalf("unmatched = %#v; want Missing Series", result.Body.Unmatched)
	}
}

func imageDataURL(mime string, data []byte) string {
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func coverPath(t *testing.T, covers *CoverCache, image string) string {
	t.Helper()
	name, ok := covers.localFileName(image)
	if !ok {
		t.Fatalf("cover URL %q is not local", image)
	}
	return filepath.Join(covers.dir, name)
}

func assertCoverMaxDimension(t *testing.T, path string, want int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open cover: %v", err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("decode cover: %v", err)
	}
	if got := max(img.Bounds().Dx(), img.Bounds().Dy()); got != want {
		t.Fatalf("cover max dimension = %d; want %d", got, want)
	}
}

func TestReadingOrderCBLExportBuildsFlatCBLXML(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)

	metronID := 98765
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, metron_issue_id)
		VALUES ('Batman & Robin', 2011, '1', 'DC Comics', '2011-09-01', ?),
			('Batman & Robin', 2011, '2', 'DC Comics', '2011-10-01', NULL);
	`, metronID); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Batman & Robin"})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	input := &SetReadingOrderComicsInput{ID: created.Body.ID}
	input.Body.Entries = []ReadingOrderEntryPayload{
		{Type: "comic", ComicID: 1},
		{Type: "comic", ComicID: 2},
	}
	if _, err := setReadingOrderComics(ctx, db, input); err != nil {
		t.Fatalf("set entries: %v", err)
	}

	result, err := exportReadingOrderCBL(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("exportReadingOrderCBL: %v", err)
	}

	for _, fragment := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<Name>Batman &amp; Robin</Name>`,
		`<NumIssues>2</NumIssues>`,
		`Series="Batman &amp; Robin" Number="1" Volume="2011" Year="2011"`,
		`<Database Name="metron" Issue="98765"></Database>`,
	} {
		if !strings.Contains(result.Body.Content, fragment) {
			t.Fatalf("export missing %q in:\n%s", fragment, result.Body.Content)
		}
	}
	if result.Body.Filename != "Batman-Robin.cbl" {
		t.Fatalf("filename = %q; want Batman-Robin.cbl", result.Body.Filename)
	}
}

func setupReadingOrderCBLTestDB(t *testing.T) *sqlx.DB {
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
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (id, name, is_default) VALUES (1, 'Default', 1);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
			CHECK (parent_reading_order_id <> child_reading_order_id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestRateReadingOrderUsesUserRatings(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (2, 'Second Reader')`); err != nil {
		t.Fatalf("insert second user: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Rated order"})
	if err != nil {
		t.Fatalf("create reading order: %v", err)
	}

	rated, err := rateReadingOrder(ctx, db, created.Body.ID, 4)
	if err != nil {
		t.Fatalf("rate reading order: %v", err)
	}
	if rated.Body.Rating != 4 || rated.Body.RatingCount != 1 {
		t.Fatalf("rating summary = %v/%d; want 4/1", rated.Body.Rating, rated.Body.RatingCount)
	}
	if rated.Body.MyRating == nil || *rated.Body.MyRating != 4 {
		t.Fatalf("my rating = %#v; want 4", rated.Body.MyRating)
	}

	secondCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := rateReadingOrder(secondCtx, db, created.Body.ID, 2); err != nil {
		t.Fatalf("rate reading order as second user: %v", err)
	}
	refreshed, err := getReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("get reading order: %v", err)
	}
	if refreshed.Body.Rating != 3 || refreshed.Body.RatingCount != 2 {
		t.Fatalf("rating summary = %v/%d; want 3/2", refreshed.Body.Rating, refreshed.Body.RatingCount)
	}
	if refreshed.Body.MyRating == nil || *refreshed.Body.MyRating != 4 {
		t.Fatalf("my rating after second user = %#v; want 4", refreshed.Body.MyRating)
	}

	cleared, err := rateReadingOrder(ctx, db, created.Body.ID, 0)
	if err != nil {
		t.Fatalf("clear reading order rating: %v", err)
	}
	if cleared.Body.Rating != 2 || cleared.Body.RatingCount != 1 {
		t.Fatalf("rating summary after clear = %v/%d; want 2/1", cleared.Body.Rating, cleared.Body.RatingCount)
	}
	if cleared.Body.MyRating != nil {
		t.Fatalf("my rating after clear = %#v; want nil", cleared.Body.MyRating)
	}
}

func TestStartReadingOrderIsPerUserAndIdempotent(t *testing.T) {
	ctx := testUserContext()
	db := setupReadingOrderCBLTestDB(t)
	if _, err := db.Exec(`INSERT INTO users (id, name) VALUES (2, 'Second Reader')`); err != nil {
		t.Fatalf("insert second user: %v", err)
	}

	created, err := createReadingOrder(ctx, db, nil, ReadingOrderPayload{Name: "Started order"})
	if err != nil {
		t.Fatalf("create reading order: %v", err)
	}
	started, err := startReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("start reading order: %v", err)
	}
	if started.Body.StartedAt == nil || *started.Body.StartedAt == "" {
		t.Fatal("startedAt is empty after starting reading order")
	}
	firstStartedAt := *started.Body.StartedAt
	startedList, err := listReadingOrders(ctx, db, &ReadingOrderListInput{Started: "true"})
	if err != nil {
		t.Fatalf("list started reading orders: %v", err)
	}
	if len(startedList.Body) != 1 || startedList.Body[0].ID != created.Body.ID {
		t.Fatalf("started reading orders = %#v; want created order", startedList.Body)
	}

	startedAgain, err := startReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("start reading order again: %v", err)
	}
	if startedAgain.Body.StartedAt == nil || *startedAgain.Body.StartedAt != firstStartedAt {
		t.Fatalf("repeated start changed startedAt from %q to %#v", firstStartedAt, startedAgain.Body.StartedAt)
	}

	secondCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	secondDetail, err := getReadingOrder(secondCtx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("get reading order as second user: %v", err)
	}
	if secondDetail.Body.StartedAt != nil {
		t.Fatalf("second user's startedAt = %#v; want nil", secondDetail.Body.StartedAt)
	}

	stopped, err := stopReadingOrder(ctx, db, created.Body.ID)
	if err != nil {
		t.Fatalf("stop reading order: %v", err)
	}
	if stopped.Body.StartedAt != nil {
		t.Fatalf("startedAt after stop = %#v; want nil", stopped.Body.StartedAt)
	}
}

func TestArcCreateEntriesFavoriteAndProgress(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_arc_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_arc_starts (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (arc_id, user_id));
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z'
				);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (name) VALUES ('Default');
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO comics (series, series_year, issue, publisher, read)
		VALUES ('Series', 2026, 1, 'Publisher', 1),
			('Series', 2026, 2, 'Publisher', 0);
		INSERT INTO user_comics (comic_id, user_id, read)
		SELECT id, (SELECT id FROM users WHERE name = 'Default'), read FROM comics;
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	created, err := createArc(ctx, db, ArcPayload{Name: "Arc", Description: "Story"})
	if err != nil {
		t.Fatalf("createArc: %v", err)
	}

	input := &SetArcComicsInput{ID: created.Body.ID}
	input.Body.Comics = []ArcComicPayload{
		{ComicID: 1, Comment: "Start"},
		{ComicID: 1, Comment: "Again"},
		{ComicID: 2, Comment: "End"},
	}
	detail, err := setArcComics(ctx, db, input)
	if err != nil {
		t.Fatalf("setArcComics: %v", err)
	}
	if len(detail.Body.Comics) != 3 {
		t.Fatalf("arc comics = %d; want duplicate-preserving count 3", len(detail.Body.Comics))
	}
	if detail.Body.Progress != float64(2)/float64(3) {
		t.Fatalf("progress = %v; want 2/3", detail.Body.Progress)
	}
	if detail.Body.Comics[1].Comment != "Again" {
		t.Fatalf("second comment = %q; want Again", detail.Body.Comics[1].Comment)
	}

	updated, err := updateArc(ctx, db, created.Body.ID, ArcPayload{Name: "Arc", Description: "Story", Favorite: true})
	if err != nil {
		t.Fatalf("updateArc: %v", err)
	}
	if !updated.Body.Favorite {
		t.Fatal("arc favorite was not saved")
	}

	list, err := listArcs(ctx, db, &ArcListInput{ComicID: 2})
	if err != nil {
		t.Fatalf("listArcs: %v", err)
	}
	if len(list.Body) != 1 || list.Body[0].ID != created.Body.ID {
		t.Fatalf("filtered arcs = %#v; want created arc", list.Body)
	}
}

func TestSeriesFavoriteAndProgress(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_series_id INTEGER,
			publisher TEXT NOT NULL DEFAULT '',
			volume INTEGER NOT NULL DEFAULT 0,
			year_end INTEGER NOT NULL DEFAULT 0,
			issue_count INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_series_name_year
		ON series(name, series_year);
		CREATE UNIQUE INDEX idx_series_metron_series_id
		ON series(metron_series_id)
		WHERE metron_series_id IS NOT NULL;
		CREATE TABLE user_series_starts (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (series_id, user_id));
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z'
				);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (name) VALUES ('Default');
		INSERT INTO comics (series, series_year, issue, publisher, read)
		VALUES ('Series', 2026, 1, 'Publisher', 1),
			('Series', 2026, 2, 'Publisher', 0);
		INSERT INTO user_comics (comic_id, user_id, read)
		SELECT id, (SELECT id FROM users WHERE name = 'Default'), read FROM comics;
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	list, err := listSeries(ctx, db, &ComicSeriesListInput{})
	if err != nil {
		t.Fatalf("listSeries: %v", err)
	}
	if len(list.Body) != 1 {
		t.Fatalf("series count = %d; want 1", len(list.Body))
	}
	if list.Body[0].Progress != 0.5 || list.Body[0].ReadCount != 1 || list.Body[0].EntryCount != 2 {
		t.Fatalf("series stats = %#v; want progress .5, read 1, entries 2", list.Body[0])
	}

	detail, err := updateSeriesFavorite(ctx, db, list.Body[0].ID, true)
	if err != nil {
		t.Fatalf("updateSeriesFavorite: %v", err)
	}
	if !detail.Body.Favorite {
		t.Fatal("series favorite was not saved")
	}
	if len(detail.Body.Comics) != 2 {
		t.Fatalf("detail comics = %d; want 2", len(detail.Body.Comics))
	}
}

func TestSeriesSyncDoesNotFailWhenPruneFails(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_series_id INTEGER,
			publisher TEXT NOT NULL DEFAULT '',
			volume INTEGER NOT NULL DEFAULT 0,
			year_end INTEGER NOT NULL DEFAULT 0,
			issue_count INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_series_name_year
		ON series(name, series_year);
		CREATE TABLE user_series_starts (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (series_id, user_id));
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
			CREATE TABLE users (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z'
				);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (name) VALUES ('Default');
		INSERT INTO series (name, series_year)
		VALUES ('Stale', 2026);
		INSERT INTO comics (series, series_year, issue, publisher)
		VALUES ('Live', 2026, 1, 'Publisher');
		CREATE TRIGGER fail_series_prune
		BEFORE DELETE ON series
		BEGIN
			SELECT RAISE(FAIL, 'prune blocked');
		END;
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	list, err := listSeries(ctx, db, &ComicSeriesListInput{})
	if err != nil {
		t.Fatalf("listSeries: %v", err)
	}
	if len(list.Body) != 2 {
		t.Fatalf("series count = %d; want live plus stale rows", len(list.Body))
	}
}

func TestDocsConfigAndRouteMetadata(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, DocsConfig())
	RegisterComicRoutes(api, nil, nil)
	RegisterSeriesRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())
	RegisterCharacterRoutes(api, nil)
	RegisterReadingOrderRoutes(api, nil, nil)
	RegisterArcRoutes(api, nil)
	RegisterDashboardRoutes(api, nil)
	RegisterStatisticsRoutes(api, nil)
	RegisterMetronRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())

	openAPI := api.OpenAPI()
	if openAPI.Info.Description == "" {
		t.Fatal("OpenAPI description is empty")
	}
	if len(openAPI.Tags) != 9 {
		t.Fatalf("len(tags) = %d; want 9", len(openAPI.Tags))
	}

	listComics := openAPI.Paths["/comics"].Get
	if len(listComics.Tags) != 1 || listComics.Tags[0] != tagComics {
		t.Fatalf("list comics tags = %#v; want Comics tag", listComics.Tags)
	}
	if _, ok := listComics.Responses["400"]; !ok {
		t.Fatal("list comics response docs missing 400 error")
	}

	accountStatistics := openAPI.Paths["/account/statistics"].Get
	if len(accountStatistics.Tags) != 1 || accountStatistics.Tags[0] != tagStatistics {
		t.Fatalf("account statistics tags = %#v; want Statistics tag", accountStatistics.Tags)
	}

	dashboard := openAPI.Paths["/dashboard"].Get
	if len(dashboard.Tags) != 1 || dashboard.Tags[0] != tagDashboard {
		t.Fatalf("dashboard tags = %#v; want Dashboard tag", dashboard.Tags)
	}

	listCharacters := openAPI.Paths["/characters"].Get
	if len(listCharacters.Tags) != 1 || listCharacters.Tags[0] != tagCharacters {
		t.Fatalf("list characters tags = %#v; want Characters tag", listCharacters.Tags)
	}

	listSeries := openAPI.Paths["/series"].Get
	if len(listSeries.Tags) != 1 || listSeries.Tags[0] != tagSeries {
		t.Fatalf("list series tags = %#v; want Series tag", listSeries.Tags)
	}

	listArcs := openAPI.Paths["/arcs"].Get
	if len(listArcs.Tags) != 1 || listArcs.Tags[0] != tagArcs {
		t.Fatalf("list arcs tags = %#v; want Arcs tag", listArcs.Tags)
	}

	searchMetronArcs := openAPI.Paths["/metron/arcs"].Get
	if len(searchMetronArcs.Tags) != 1 || searchMetronArcs.Tags[0] != tagMetron {
		t.Fatalf("search Metron arcs tags = %#v; want Metron tag", searchMetronArcs.Tags)
	}

	importSeries := openAPI.Paths["/metron/series/{id}/import"].Post
	if len(importSeries.Tags) != 1 || importSeries.Tags[0] != tagMetron {
		t.Fatalf("import series tags = %#v; want Metron tag", importSeries.Tags)
	}
	if _, ok := importSeries.Responses["429"]; !ok {
		t.Fatal("import series response docs missing 429 error")
	}
}

func TestMultiUserSetupSetsSessionCookieForProtectedRoutes(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)
	RegisterComicRoutes(api, db, nil)

	setup := httptest.NewRecorder()
	setupBody := strings.NewReader(`{"mode":"multi","name":"Test","email":"test@example.com","password":"secret1"}`)
	setupReq := httptest.NewRequest(http.MethodPost, "/api/auth/setup", setupBody)
	setupReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(setup, setupReq)
	if setup.Code != http.StatusOK {
		t.Fatalf("setup status = %d; want 200: %s", setup.Code, setup.Body.String())
	}
	cookies := setup.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || cookies[0].Value == "" {
		t.Fatalf("setup cookies = %#v; want %s session cookie", cookies, sessionCookieName)
	}

	statusRecorder := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	statusReq.AddCookie(cookies[0])
	router.ServeHTTP(statusRecorder, statusReq)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status with cookie status = %d; want 200: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	var status UserStatus
	if err := json.NewDecoder(statusRecorder.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.User == nil || !status.User.IsAdmin {
		t.Fatalf("status user = %#v; want admin user", status.User)
	}
	if !status.MetronPermissions.Allowed || !metronScopeAllowed(status.MetronPermissions.Scopes, metronScopeMonitor) {
		t.Fatalf("status metron permissions = %#v; want monitor access for admin", status.MetronPermissions)
	}

	withoutCookie := httptest.NewRecorder()
	router.ServeHTTP(withoutCookie, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if withoutCookie.Code != http.StatusUnauthorized {
		t.Fatalf("comics without cookie status = %d; want 401", withoutCookie.Code)
	}

	withCookie := httptest.NewRecorder()
	comicsReq := httptest.NewRequest(http.MethodGet, "/api/comics", nil)
	comicsReq.AddCookie(cookies[0])
	router.ServeHTTP(withCookie, comicsReq)
	if withCookie.Code != http.StatusOK {
		t.Fatalf("comics with cookie status = %d; want 200: %s", withCookie.Code, withCookie.Body.String())
	}
	if !strings.Contains(withCookie.Body.String(), `"series":"Amazing Spider-Man"`) {
		t.Fatalf("comics body = %s; want seeded comic", withCookie.Body.String())
	}
}

func TestLoginRateLimitReturnsTooManyRequests(t *testing.T) {
	previousLimiter := authLoginLimiter
	authLoginLimiter = newLoginRateLimiter(loginRateLimitMaxAttempts, loginRateLimitWindow)
	t.Cleanup(func() {
		authLoginLimiter = previousLimiter
	})

	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed login user: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)

	for i := 0; i < loginRateLimitMaxAttempts; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.44:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = 429; want not rate-limited yet", i+1)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.44:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited login status = %d; want 429: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginUsesEmailAddress(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed login user: %v", err)
	}

	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Name:     "Test",
		Password: "secret1",
	}); err == nil {
		t.Fatal("loginUser accepted a username without an email")
	}

	output, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Email:    "Test@Example.com",
		Password: "secret1",
	})
	if err != nil {
		t.Fatalf("loginUser with email: %v", err)
	}
	if output.Body.User == nil || output.Body.User.Email != "test@example.com" {
		t.Fatalf("user = %#v; want logged-in user by normalized email", output.Body.User)
	}
}

func TestRegistrationRateLimitReturnsTooManyRequests(t *testing.T) {
	previousLimiter := authRegistrationLimiter
	authRegistrationLimiter = newLoginRateLimiter(registrationRateLimitMaxAttempts, registrationRateLimitWindow)
	t.Cleanup(func() {
		authRegistrationLimiter = previousLimiter
	})

	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO app_settings (key, value) VALUES ('registration_mode', 'open');
	`); err != nil {
		t.Fatalf("seed open registration mode: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)

	for i := 0; i < registrationRateLimitMaxAttempts; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.45:1234"
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d status = 429; want not rate-limited yet", i+1)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"name":"","password":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.45:1234"
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("rate-limited registration status = %d; want 429: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRequireAdminUserFailsWithoutUserContext(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	if userID, err := requireAdminUser(context.Background(), db); err == nil {
		t.Fatalf("requireAdminUser without user context = %d, nil; want error", userID)
	}
}

func TestPerUserEndpointFailsWithoutUserContext(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterComicRoutes(api, db, nil)

	recorder := httptest.NewRecorder()
	body := strings.NewReader(`{"read":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/comic/1/read", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("read-status without user context status = %d; want 401: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateAccountRenamesAndRequiresCurrentPassword(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
	`, hash); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := updateAccount(ctx, db, UpdateAccountPayload{
		Name:            "Renamed",
		CurrentPassword: "wrong",
		NewPassword:     "secret2",
	}); err == nil {
		t.Fatal("updateAccount accepted an incorrect current password")
	}

	output, err := updateAccount(ctx, db, UpdateAccountPayload{
		Name:            "Renamed",
		CurrentPassword: "secret1",
		NewPassword:     "secret2",
	})
	if err != nil {
		t.Fatalf("updateAccount: %v", err)
	}
	if output.Body.User == nil || output.Body.User.Name != "Renamed" {
		t.Fatalf("user = %#v; want renamed current user", output.Body.User)
	}

	var newHash string
	if err := db.Get(&newHash, `SELECT password_hash FROM users WHERE id = 1`); err != nil {
		t.Fatalf("fetch password hash: %v", err)
	}
	if !checkPassword("secret2", newHash) {
		t.Fatal("new password hash does not match updated password")
	}
}

func TestDeleteAccountRequiresPasswordAndAnotherAdmin(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
		INSERT INTO users (id, name, email, password_hash) VALUES (2, 'Other', 'other@example.com', 'hash');
		INSERT INTO user_sessions (token, user_id) VALUES ('session-1', 1);
		INSERT INTO reading_orders (name, author_user_id) VALUES ('Mine', 1);
	`, hash); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}

	ctx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "wrong"}); err == nil {
		t.Fatal("deleteAccount accepted an incorrect current password")
	}
	if _, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "secret1"}); err == nil {
		t.Fatal("deleteAccount deleted the only admin account")
	}

	if _, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE id = 2`); err != nil {
		t.Fatalf("promote other account: %v", err)
	}
	output, err := deleteAccount(ctx, db, DeleteAccountPayload{CurrentPassword: "secret1"})
	if err != nil {
		t.Fatalf("deleteAccount: %v", err)
	}
	if output.Body.User != nil || output.Body.Mode != userModeMulti {
		t.Fatalf("status = %#v; want logged-out multi-user status", output.Body)
	}
	if len(output.SetCookie) != 1 || output.SetCookie[0].MaxAge >= 0 {
		t.Fatalf("cookies = %#v; want expired session cookie", output.SetCookie)
	}

	var userCount int
	if err := db.Get(&userCount, `SELECT COUNT(*) FROM users WHERE id = 1`); err != nil {
		t.Fatalf("count deleted user: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("deleted user count = %d; want 0", userCount)
	}
	var sessionCount int
	if err := db.Get(&sessionCount, `SELECT COUNT(*) FROM user_sessions WHERE user_id = 1`); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("deleted user's sessions = %d; want 0", sessionCount)
	}
	var authorCount int
	if err := db.Get(&authorCount, `SELECT COUNT(*) FROM reading_orders WHERE author_user_id = 1`); err != nil {
		t.Fatalf("count authored orders: %v", err)
	}
	if authorCount != 0 {
		t.Fatalf("authored orders = %d; want 0", authorCount)
	}
}

func TestAdminCanDeleteNonAdminUser(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0);
		INSERT INTO user_sessions (token, user_id) VALUES ('reader-session', 2);
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit) VALUES (2, 1, '*', 0);
		INSERT INTO user_metron_request_log (user_id, scope, endpoint) VALUES (2, 'search', '/issue/');
		INSERT INTO user_comics (comic_id, user_id, read) VALUES (1, 2, 1);
		INSERT INTO reading_orders (name, author_user_id) VALUES ('Reader list', 2);
	`); err != nil {
		t.Fatalf("seed reader account: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteUser(adminCtx, db, 2); err != nil {
		t.Fatalf("deleteUser: %v", err)
	}

	for table, query := range map[string]string{
		"users":                   `SELECT COUNT(*) FROM users WHERE id = 2`,
		"user_sessions":           `SELECT COUNT(*) FROM user_sessions WHERE user_id = 2`,
		"user_metron_permissions": `SELECT COUNT(*) FROM user_metron_permissions WHERE user_id = 2`,
		"user_metron_request_log": `SELECT COUNT(*) FROM user_metron_request_log WHERE user_id = 2`,
		"user_comics":             `SELECT COUNT(*) FROM user_comics WHERE user_id = 2`,
	} {
		var count int
		if err := db.Get(&count, query); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d; want 0", table, count)
		}
	}
	var authorCount int
	if err := db.Get(&authorCount, `SELECT COUNT(*) FROM reading_orders WHERE author_user_id = 2`); err != nil {
		t.Fatalf("count reading order authors: %v", err)
	}
	if authorCount != 0 {
		t.Fatalf("reading order author count = %d; want 0", authorCount)
	}
}

func TestNonAdminCannotDeleteUsers(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0);
		INSERT INTO users (id, name, is_admin) VALUES (3, 'Other', 0);
	`); err != nil {
		t.Fatalf("seed accounts: %v", err)
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := deleteUser(readerCtx, db, 3); err == nil {
		t.Fatal("deleteUser by non-admin returned nil error")
	}
}

func TestDeleteUserRejectsOnlyAdmin(t *testing.T) {
	db := setupMountedAuthTestDB(t)

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := deleteUser(adminCtx, db, 1); err == nil {
		t.Fatal("deleteUser deleted the only admin account")
	}
}

func TestMetronPermissionsControlScopesAndHourlyLimit(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/comics"); err == nil {
		t.Fatal("authorizeMetron returned nil for reader without Metron permissions")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateUserMetronPermissions(adminCtx, db, 2, UserMetronPermissions{
		Allowed:     true,
		Scopes:      []string{metronScopeSearch},
		HourlyLimit: 1,
	})
	if err != nil {
		t.Fatalf("updateUserMetronPermissions: %v", err)
	}
	if !output.Body.MetronPermissions.Allowed || output.Body.MetronPermissions.HourlyLimit != 1 {
		t.Fatalf("permissions = %#v; want allowed with hourly limit 1", output.Body.MetronPermissions)
	}

	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/comics"); err != nil {
		t.Fatalf("authorize search: %v", err)
	}
	if err := authorizeMetron(readerCtx, db, metronScopeImport, "POST /metron/comics/{id}/import"); err == nil {
		t.Fatal("authorize import returned nil for search-only user")
	}
	if err := authorizeMetron(readerCtx, db, metronScopeSearch, "GET /metron/series"); err == nil {
		t.Fatal("authorize search returned nil after hourly limit was reached")
	}
}

func TestAdminCanPromoteOtherUsers(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateUserAdmin(adminCtx, db, 2, UpdateUserAdminPayload{IsAdmin: true})
	if err != nil {
		t.Fatalf("updateUserAdmin promote: %v", err)
	}
	if !output.Body.User.IsAdmin {
		t.Fatalf("promoted user isAdmin = false; want true")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updateUserAdmin(readerCtx, db, 1, UpdateUserAdminPayload{IsAdmin: false}); err != nil {
		t.Fatalf("promoted user should be able to update admin roles: %v", err)
	}
	if _, err := updateUserAdmin(readerCtx, db, 2, UpdateUserAdminPayload{IsAdmin: false}); err == nil {
		t.Fatal("updateUserAdmin allowed current user to remove own admin role")
	}
}

func TestListUsersIncludesAccountTimestamps(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name, email, email_verified_at, is_admin, created_at)
		VALUES (2, 'Reader', 'reader@example.com', '2026-07-10 11:00:00', 0, '2026-07-09 10:00:00')
	`); err != nil {
		t.Fatalf("create reader user: %v", err)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := listUsers(adminCtx, db)
	if err != nil {
		t.Fatalf("listUsers: %v", err)
	}

	for _, entry := range output.Body {
		if entry.User.ID != 2 {
			continue
		}
		if entry.User.CreatedAt != "2026-07-09 10:00:00" {
			t.Fatalf("createdAt = %q; want seeded timestamp", entry.User.CreatedAt)
		}
		if !entry.User.EmailVerified || entry.User.EmailVerifiedAt != "2026-07-10 11:00:00" {
			t.Fatalf("email verification = (%v, %q); want verified timestamp", entry.User.EmailVerified, entry.User.EmailVerifiedAt)
		}
		return
	}

	t.Fatal("reader missing from users response")
}

func TestRegisterUserRequiresValidInvite(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}

	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "No Invite",
		Email:                "no-invite@example.com",
		EmailConfirmation:    "no-invite@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	invite, err := createUserInvite(adminCtx, db)
	if err != nil {
		t.Fatalf("createUserInvite: %v", err)
	}
	if invite.Body.Token == "" || invite.Body.ExpiresAt == "" {
		t.Fatalf("invite = %#v; want token and expiry", invite.Body)
	}

	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Missing Email Confirmation",
		Email:                "missing-email-confirmation@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted missing email confirmation")
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Password Mismatch",
		Email:                "password-mismatch@example.com",
		EmailConfirmation:    "password-mismatch@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret2",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted mismatched password confirmation")
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Invited",
		Email:                "invited@example.com",
		EmailConfirmation:    "invited@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err != nil {
		t.Fatalf("registerUser with invite: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Reuse",
		Email:                "reuse@example.com",
		EmailConfirmation:    "reuse@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          invite.Body.Token,
	}); err == nil {
		t.Fatal("registerUser accepted a used invite")
	}

	if _, err := db.Exec(`
		INSERT INTO user_invites (token, expires_at)
		VALUES ('expired-token', ?)
	`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed expired invite: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Expired",
		Email:                "expired@example.com",
		EmailConfirmation:    "expired@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
		InviteToken:          "expired-token",
	}); err == nil {
		t.Fatal("registerUser accepted an expired invite")
	}
}

func TestRegistrationModeDefaultsAndAdminCanUpdate(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)`); err != nil {
		t.Fatalf("seed reader user: %v", err)
	}

	mode, err := registrationMode(context.Background(), db)
	if err != nil {
		t.Fatalf("registrationMode default: %v", err)
	}
	if mode != registrationModeInviteOnly {
		t.Fatalf("registrationMode default = %q; want %q", mode, registrationModeInviteOnly)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Default Blocked",
		Email:                "default-blocked@example.com",
		EmailConfirmation:    "default-blocked@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite succeeded while registration mode is unset")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updateRegistrationMode(readerCtx, db, UpdateRegistrationModePayload{Mode: registrationModeOpen}); err == nil {
		t.Fatal("non-admin updateRegistrationMode returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updateRegistrationMode(adminCtx, db, UpdateRegistrationModePayload{Mode: registrationModeOpen})
	if err != nil {
		t.Fatalf("updateRegistrationMode: %v", err)
	}
	if output.Body.RegistrationMode != registrationModeOpen {
		t.Fatalf("output registrationMode = %q; want %q", output.Body.RegistrationMode, registrationModeOpen)
	}
	mode, err = registrationMode(context.Background(), db)
	if err != nil {
		t.Fatalf("registrationMode after update: %v", err)
	}
	if mode != registrationModeOpen {
		t.Fatalf("registrationMode after update = %q; want %q", mode, registrationModeOpen)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Signup",
		Email:                "open-signup@example.com",
		EmailConfirmation:    "open-signup@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err != nil {
		t.Fatalf("registerUser without invite in open mode: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Mismatch",
		Email:                "open-mismatch@example.com",
		EmailConfirmation:    "other@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser accepted mismatched email confirmation in open mode")
	}

	if _, err := updateRegistrationMode(adminCtx, db, UpdateRegistrationModePayload{Mode: registrationModeInviteOnly}); err != nil {
		t.Fatalf("updateRegistrationMode back to invite_only: %v", err)
	}
	if _, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Invite Blocked",
		Email:                "invite-blocked@example.com",
		EmailConfirmation:    "invite-blocked@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	}); err == nil {
		t.Fatal("registerUser without invite succeeded after switching back to invite_only")
	}
}

func TestOpenRegistrationRequiresEmailVerificationBeforeAccess(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO app_settings (key, value) VALUES ('registration_mode', 'open');
	`); err != nil {
		t.Fatalf("seed open registration mode: %v", err)
	}

	output, err := registerUser(context.Background(), db, UserCredentialsPayload{
		Name:                 "Open Pending",
		Email:                "open-pending@example.com",
		EmailConfirmation:    "open-pending@example.com",
		Password:             "secret1",
		PasswordConfirmation: "secret1",
	})
	if err != nil {
		t.Fatalf("registerUser open: %v", err)
	}
	if !output.Body.EmailVerificationRequired || output.Body.EmailVerificationEmail != "open-pending@example.com" {
		t.Fatalf("registration status = %#v; want email verification required", output.Body)
	}
	if len(output.SetCookie) != 0 {
		t.Fatalf("registration cookies = %#v; want no session before verification", output.SetCookie)
	}

	var row struct {
		ID              int    `db:"id"`
		EmailVerifiedAt string `db:"email_verified_at"`
	}
	if err := db.Get(&row, `SELECT id, email_verified_at FROM users WHERE email = 'open-pending@example.com'`); err != nil {
		t.Fatalf("fetch registered user: %v", err)
	}
	if row.EmailVerifiedAt != "" {
		t.Fatalf("email_verified_at = %q; want empty before verification", row.EmailVerifiedAt)
	}
	var tokenCount int
	if err := db.Get(&tokenCount, `SELECT COUNT(*) FROM user_email_verifications WHERE user_id = ? AND used_at = ''`, row.ID); err != nil {
		t.Fatalf("count verification tokens: %v", err)
	}
	if tokenCount != 1 {
		t.Fatalf("active verification token count = %d; want 1", tokenCount)
	}

	loginOutput, err := loginUser(context.Background(), db, UserCredentialsPayload{
		Email:    "open-pending@example.com",
		Password: "secret1",
	})
	if err != nil {
		t.Fatalf("loginUser pending: %v", err)
	}
	if !loginOutput.Body.EmailVerificationRequired || len(loginOutput.SetCookie) != 0 {
		t.Fatalf("pending login = %#v cookies %#v; want verification required without session", loginOutput.Body, loginOutput.SetCookie)
	}

	token, _, err := createEmailVerification(context.Background(), db, row.ID)
	if err != nil {
		t.Fatalf("create verification token: %v", err)
	}
	verified, err := verifyEmail(context.Background(), db, token)
	if err != nil {
		t.Fatalf("verifyEmail: %v", err)
	}
	if verified.Body.User == nil || !verified.Body.User.EmailVerified {
		t.Fatalf("verified user = %#v; want verified session user", verified.Body.User)
	}
	if len(verified.SetCookie) != 1 || verified.SetCookie[0].Value == "" {
		t.Fatalf("verification cookies = %#v; want session", verified.SetCookie)
	}
}

func TestPasswordResetChangesPasswordAndConsumesToken(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	db := setupMountedAuthTestDB(t)
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		UPDATE users SET name = 'Test', email = 'test@example.com', password_hash = ? WHERE id = 1;
		INSERT INTO user_sessions (token, user_id, expires_at) VALUES ('old-session', 1, '2999-01-01T00:00:00Z');
	`, hash); err != nil {
		t.Fatalf("seed reset user: %v", err)
	}

	if _, err := requestPasswordReset(context.Background(), db, ForgotPasswordPayload{Email: "test@example.com"}); err != nil {
		t.Fatalf("requestPasswordReset: %v", err)
	}
	var userID int
	if err := db.Get(&userID, `SELECT id FROM users WHERE email = 'test@example.com'`); err != nil {
		t.Fatalf("fetch user id: %v", err)
	}
	var tokenCount int
	if err := db.Get(&tokenCount, `SELECT COUNT(*) FROM user_password_resets WHERE user_id = ? AND used_at = ''`, userID); err != nil {
		t.Fatalf("count reset tokens: %v", err)
	}
	if tokenCount != 1 {
		t.Fatalf("active reset token count = %d; want 1", tokenCount)
	}

	token, _, err := createPasswordReset(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("createPasswordReset: %v", err)
	}
	if _, err := resetPassword(context.Background(), db, ResetPasswordPayload{
		Token:                token,
		Password:             "secret2",
		PasswordConfirmation: "secret2",
	}); err != nil {
		t.Fatalf("resetPassword: %v", err)
	}
	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{Email: "test@example.com", Password: "secret1"}); err == nil {
		t.Fatal("loginUser accepted old password after reset")
	}
	if _, err := loginUser(context.Background(), db, UserCredentialsPayload{Email: "test@example.com", Password: "secret2"}); err != nil {
		t.Fatalf("loginUser with new password: %v", err)
	}
	if _, err := resetPassword(context.Background(), db, ResetPasswordPayload{
		Token:                token,
		Password:             "secret3",
		PasswordConfirmation: "secret3",
	}); err == nil {
		t.Fatal("resetPassword accepted a used token")
	}
	var sessionCount int
	if err := db.Get(&sessionCount, `SELECT COUNT(*) FROM user_sessions WHERE token = 'old-session'`); err != nil {
		t.Fatalf("count old sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("old session count = %d; want 0", sessionCount)
	}
}

func TestPublicAccessDefaultsAndAdminCanUpdate(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi')`); err != nil {
		t.Fatalf("seed multi-user mode: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, is_admin) VALUES (2, 'Reader', 0)`); err != nil {
		t.Fatalf("seed reader user: %v", err)
	}

	enabled, err := publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled default: %v", err)
	}
	if enabled {
		t.Fatal("publicAccessEnabled default = true; want false")
	}

	readerCtx := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if _, err := updatePublicAccess(readerCtx, db, UpdatePublicAccessPayload{Enabled: true}); err == nil {
		t.Fatal("non-admin updatePublicAccess returned nil error")
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	output, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: true})
	if err != nil {
		t.Fatalf("updatePublicAccess enable: %v", err)
	}
	if !output.Body.PublicAccess {
		t.Fatalf("output publicAccess = false; want true")
	}
	enabled, err = publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled after enable: %v", err)
	}
	if !enabled {
		t.Fatal("publicAccessEnabled after enable = false; want true")
	}

	if _, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: false}); err != nil {
		t.Fatalf("updatePublicAccess disable: %v", err)
	}
	enabled, err = publicAccessEnabled(context.Background(), db)
	if err != nil {
		t.Fatalf("publicAccessEnabled after disable: %v", err)
	}
	if enabled {
		t.Fatal("publicAccessEnabled after disable = true; want false")
	}
}

func TestPublicAccessAllowsAnonymousReadOnlyRoutes(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO app_settings (key, value) VALUES ('user_mode', 'multi');
		INSERT INTO reading_orders (id, name, author_user_id) VALUES (1, 'Public Order', 1);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position) VALUES (1, 1, 1);
	`); err != nil {
		t.Fatalf("seed public library data: %v", err)
	}

	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	apiRouter.Use(UserMiddleware(db))
	router.Mount("/api", apiRouter)
	api := humachi.New(apiRouter, DocsConfig())
	RegisterUserRoutes(api, db)
	RegisterComicRoutes(api, db, nil)
	RegisterReadingOrderRoutes(api, db, nil)

	disabled := httptest.NewRecorder()
	router.ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if disabled.Code != http.StatusUnauthorized {
		t.Fatalf("public disabled comics status = %d; want 401", disabled.Code)
	}

	adminCtx := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	if _, err := updatePublicAccess(adminCtx, db, UpdatePublicAccessPayload{Enabled: true}); err != nil {
		t.Fatalf("enable public access: %v", err)
	}

	comics := httptest.NewRecorder()
	router.ServeHTTP(comics, httptest.NewRequest(http.MethodGet, "/api/comics", nil))
	if comics.Code != http.StatusOK {
		t.Fatalf("public comics status = %d; want 200: %s", comics.Code, comics.Body.String())
	}
	if !strings.Contains(comics.Body.String(), `"read":true`) {
		t.Fatalf("public comics body = %s; want default user's read status", comics.Body.String())
	}

	order := httptest.NewRecorder()
	router.ServeHTTP(order, httptest.NewRequest(http.MethodGet, "/api/readingOrders/1", nil))
	if order.Code != http.StatusOK {
		t.Fatalf("public reading order status = %d; want 200: %s", order.Code, order.Body.String())
	}
	if strings.Contains(order.Body.String(), `"canEdit":true`) {
		t.Fatalf("public reading order body = %s; want canEdit false", order.Body.String())
	}

	cbl := httptest.NewRecorder()
	router.ServeHTTP(cbl, httptest.NewRequest(http.MethodGet, "/api/readingOrders/1/cbl", nil))
	if cbl.Code != http.StatusOK {
		t.Fatalf("public CBL export status = %d; want 200: %s", cbl.Code, cbl.Body.String())
	}
	if !strings.Contains(cbl.Body.String(), "Public Order") {
		t.Fatalf("public CBL export body = %s; want reading order name", cbl.Body.String())
	}

	mutate := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/comic/1/read", strings.NewReader(`{"read":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(mutate, req)
	if mutate.Code != http.StatusUnauthorized {
		t.Fatalf("public mutation status = %d; want 401", mutate.Code)
	}
}

func TestExpiredSessionTokenIsRejected(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO user_sessions (token, user_id, expires_at)
		VALUES ('expired-session', 1, ?)
	`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatalf("seed expired session: %v", err)
	}

	if userID, err := userIDFromSessionToken(context.Background(), db, "expired-session"); err == nil {
		t.Fatalf("expired session returned user %d, nil error; want error", userID)
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM user_sessions WHERE token = 'expired-session'`); err != nil {
		t.Fatalf("count expired sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expired session count = %d; want 0", count)
	}
}

func TestSessionCookiesDefaultToLocalHTTPAndAreConfigurable(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	t.Setenv("COOKIE_SECURE", "")

	cookie, err := createSession(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false by default for local HTTP")
	}
	if expired := expiredSessionCookie(context.Background()); expired.Secure {
		t.Fatal("expired session cookie Secure = true; want false by default for local HTTP")
	}

	t.Setenv("COOKIE_SECURE", "true")
	cookie, err = createSession(context.Background(), db, 1)
	if err != nil {
		t.Fatalf("createSession with COOKIE_SECURE=true: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("session cookie Secure = false; want true when COOKIE_SECURE=true")
	}
	if expired := expiredSessionCookie(context.Background()); !expired.Secure {
		t.Fatal("expired session cookie Secure = false; want true when COOKIE_SECURE=true")
	}
}

func TestSessionCookiesAutoUpgradeBehindHTTPSReverseProxy(t *testing.T) {
	db := setupMountedAuthTestDB(t)
	t.Setenv("COOKIE_SECURE", "")

	plainCtx := context.WithValue(context.Background(), contextSecureRequestKey{}, false)
	cookie, err := createSession(plainCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false when the request arrived over plain HTTP")
	}

	httpsCtx := context.WithValue(context.Background(), contextSecureRequestKey{}, true)
	cookie, err = createSession(httpsCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession behind HTTPS proxy: %v", err)
	}
	if !cookie.Secure {
		t.Fatal("session cookie Secure = false; want true when X-Forwarded-Proto/TLS indicated HTTPS")
	}

	// An explicit COOKIE_SECURE still wins over what the request looked like.
	t.Setenv("COOKIE_SECURE", "false")
	cookie, err = createSession(httpsCtx, db, 1)
	if err != nil {
		t.Fatalf("createSession with COOKIE_SECURE=false: %v", err)
	}
	if cookie.Secure {
		t.Fatal("session cookie Secure = true; want false when COOKIE_SECURE=false overrides detection")
	}
}

func TestPasswordHashUsesCurrentIterationsAndVerifiesOldHashes(t *testing.T) {
	hash, err := hashPassword("secret1")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[1] != "600000" {
		t.Fatalf("hash = %q; want current iteration count encoded", hash)
	}
	if !checkPassword("secret1", hash) {
		t.Fatal("checkPassword rejected current hash")
	}

	salt := []byte("0123456789abcdef")
	key := derivePasswordKey([]byte("secret1"), salt, 120000, 32)
	oldHash := "pbkdf2_sha256$120000$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key)
	if !checkPassword("secret1", oldHash) {
		t.Fatal("checkPassword rejected old iteration count")
	}
	if !passwordHashNeedsUpgrade(oldHash) {
		t.Fatal("passwordHashNeedsUpgrade returned false for old hash")
	}
}

func setupMountedAuthTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_issue_id INTEGER,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT ''
		);
			CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
					email TEXT NOT NULL DEFAULT '',
					email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
					password_hash TEXT NOT NULL DEFAULT '',
				is_default INTEGER NOT NULL DEFAULT 0,
				is_admin INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_reading_list_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE TABLE user_reading_orders (
			reading_order_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_ratings (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			rating REAL NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (reading_order_id, user_id)
		);
		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (reading_order_id, comic_id, position)
		);
		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id, position)
		);
		CREATE TABLE user_sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TEXT NOT NULL DEFAULT ''
		);
			CREATE TABLE user_invites (
				token TEXT PRIMARY KEY,
				created_by_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
				expires_at TEXT NOT NULL DEFAULT '',
				used_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE user_email_verifications (
				token_hash TEXT PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				expires_at TEXT NOT NULL,
				used_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE user_password_resets (
				token_hash TEXT PRIMARY KEY,
				user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				expires_at TEXT NOT NULL,
				used_at TEXT NOT NULL DEFAULT '',
				created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			skipped INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		CREATE TABLE user_metron_permissions (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			allowed INTEGER NOT NULL DEFAULT 0,
			scopes TEXT NOT NULL DEFAULT '',
			hourly_limit INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE user_metron_request_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			scope TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO users (id, name, is_default, is_admin) VALUES (1, 'Default', 1, 1);
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit) VALUES (1, 1, '*', 0);
		INSERT INTO comics (series, series_year, issue, publisher)
		VALUES ('Amazing Spider-Man', 1963, '1', 'Marvel');
		INSERT INTO user_comics (comic_id, user_id, read) VALUES (1, 1, 1);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestMountedDocsLoadMountedOpenAPISpec(t *testing.T) {
	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	router.Mount("/api", apiRouter)
	humachi.New(apiRouter, DocsConfig())

	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	if docs.Code != http.StatusOK {
		t.Fatalf("docs status = %d; want 200", docs.Code)
	}
	if !strings.Contains(docs.Body.String(), `apiDescriptionUrl="/api/openapi.yaml"`) {
		t.Fatalf("docs body did not reference mounted OpenAPI spec: %s", docs.Body.String())
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("spec status = %d; want 200", spec.Code)
	}
}
