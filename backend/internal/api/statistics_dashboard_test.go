package api

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestUserStatisticsAndAchievements(t *testing.T) {
	ctx := testUserContext()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
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
			metron_synced_at TEXT NOT NULL DEFAULT '',
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
			is_public INTEGER NOT NULL DEFAULT 1,
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
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
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
		CREATE TABLE user_arcs (
			arc_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (arc_id, user_id)
		);
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE user_series (
			series_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (series_id, user_id)
		);
		CREATE TABLE characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_characters (
			character_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
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
		INSERT INTO user_reading_orders (reading_order_id, user_id, started_at) VALUES (1, 1, CURRENT_TIMESTAMP);
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position)
		VALUES (1, 1, 1), (1, 2, 2), (2, 3, 1);
		INSERT INTO arcs (id, name) VALUES (1, 'Alpha arc'), (2, 'Beta arc');
		INSERT INTO user_arcs (arc_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:30:00Z');
		INSERT INTO arc_comics (arc_id, comic_id, position)
		VALUES (1, 1, 1), (1, 2, 2), (2, 3, 1);
		INSERT INTO user_series (series_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:45:00Z');
		INSERT INTO characters (id, name) VALUES (1, 'Hero'), (2, 'Sidekick'), (3, 'Cameo');
		INSERT INTO user_characters (character_id, user_id, started_at) VALUES (1, 1, '2026-07-01T09:50:00Z');
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
		_ = db.Close()
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
			description TEXT NOT NULL DEFAULT '',
			metron_synced_at TEXT NOT NULL DEFAULT ''
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
			is_public INTEGER NOT NULL DEFAULT 1,
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
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
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
		CREATE TABLE reading_order_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reading_order_id INTEGER NOT NULL,
			position INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_arcs (
			arc_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
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
		CREATE TABLE user_characters (
			character_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (character_id, user_id)
		);
		CREATE TABLE comic_characters (
			comic_id INTEGER NOT NULL,
			character_id INTEGER NOT NULL,
			PRIMARY KEY (comic_id, character_id)
		);
		CREATE TABLE character_collections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at TEXT
		);
		CREATE TABLE character_collection_members (
			collection_id INTEGER NOT NULL,
			character_id INTEGER NOT NULL,
			PRIMARY KEY (collection_id, character_id)
		);
		CREATE TABLE user_series (
			series_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (series_id, user_id)
		);

		INSERT INTO users (id, name, is_default) VALUES (1, 'Reader', 1);
		INSERT INTO series (id, name, series_year) VALUES (1, 'Alpha', 2020);
		INSERT INTO comics (id, series_id, series, series_year, issue, publisher)
		VALUES (1, 1, 'Alpha', 2020, '1', 'Pub'), (2, 1, 'Alpha', 2020, '2', 'Pub');
		INSERT INTO reading_orders (id, name, author_user_id) VALUES
			(1, 'Alpha order', 1),
			(2, 'Favorite only', 1);
		INSERT INTO user_reading_orders (reading_order_id, user_id, started_at)
		VALUES (1, 1, '2026-07-01T10:00:00Z'), (2, 1, NULL);
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
	t.Cleanup(func() { _ = db.Close() })

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
		CREATE TABLE user_arcs (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE characters (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE comic_characters (comic_id INTEGER NOT NULL, character_id INTEGER NOT NULL);
		CREATE TABLE user_characters (character_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE user_series (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0);

		INSERT INTO series (id, name, series_year) VALUES
			(1, 'Release order', 2020),
			(2, 'Empty series', 2020);
		INSERT INTO comics (id, series_id, series, series_year, issue, cover_date) VALUES
			(1, 1, 'Release order', 2020, '1', '2026-03-01'),
			(2, 1, 'Release order', 2020, '2', '2026-01-01'),
			(3, 1, 'Release order', 2020, '3', '2026-02-01');
		INSERT INTO arcs (id, name) VALUES (1, 'Arc'), (2, 'Empty arc');
		INSERT INTO arc_comics (arc_id, comic_id, position) VALUES (1, 1, 1), (1, 2, 3), (1, 3, 2);
		INSERT INTO user_arcs (arc_id, user_id, started_at) VALUES
			(1, 1, '2026-01-01'),
			(2, 1, '2026-01-02');
		INSERT INTO characters (id, name) VALUES (1, 'Hero');
		INSERT INTO comic_characters (comic_id, character_id) VALUES (1, 1), (2, 1), (3, 1);
		INSERT INTO user_characters (character_id, user_id, started_at) VALUES (1, 1, '2026-01-01');
		INSERT INTO user_series (series_id, user_id, started_at) VALUES
			(1, 1, '2026-01-01'),
			(2, 1, '2026-01-02');
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	loaders := []struct {
		name             string
		load             func(context.Context, *sqlx.DB, int) ([]DashboardItem, error)
		expectsEmptyItem bool
	}{
		{name: "arc", load: dashboardArcs, expectsEmptyItem: true},
		{name: "character", load: dashboardCharacters},
		{name: "series", load: dashboardSeries, expectsEmptyItem: true},
	}
	for _, loader := range loaders {
		t.Run(loader.name, func(t *testing.T) {
			items, err := loader.load(ctx, db, 1)
			if err != nil {
				t.Fatalf("load dashboard items: %v", err)
			}
			if len(items) == 0 || items[0].NextComic == nil {
				t.Fatalf("items = %#v; want an item with a next comic", items)
			}
			if got := items[0].NextComic.ID; got != 2 {
				t.Fatalf("next comic = %d; want earliest release-date comic 2", got)
			}
			if loader.expectsEmptyItem {
				if len(items) != 2 {
					t.Fatalf("items = %#v; want populated and empty started items", items)
				}
				if items[1].NextComic != nil || items[1].Progress != 0 {
					t.Fatalf("empty item = %#v; want no next comic and zero progress", items[1])
				}
			}
		})
	}
}
