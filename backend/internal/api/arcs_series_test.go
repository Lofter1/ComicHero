package api

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

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
		CREATE TABLE user_arcs (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (arc_id, user_id));
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
		CREATE TABLE user_series (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (series_id, user_id));
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
		CREATE TABLE user_series (series_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (series_id, user_id));
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
