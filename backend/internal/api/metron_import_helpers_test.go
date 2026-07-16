package api

import (
	"net/http"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
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
			is_public INTEGER NOT NULL DEFAULT 1,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			metron_reading_list_id INTEGER,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE UNIQUE INDEX idx_reading_orders_metron_reading_list_id
		ON reading_orders(metron_reading_list_id)
		WHERE metron_reading_list_id IS NOT NULL;
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
			metron_issue_id INTEGER,
			comic_vine_id INTEGER,
			metron_synced_at TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_comics_metron_issue_id
		ON comics(metron_issue_id)
		WHERE metron_issue_id IS NOT NULL;
		CREATE UNIQUE INDEX idx_comics_comic_vine_id
		ON comics(comic_vine_id)
		WHERE comic_vine_id IS NOT NULL;

		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL DEFAULT '',
			email_verified_at TEXT NOT NULL DEFAULT '2026-01-01T00:00:00Z',
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

		CREATE TABLE characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_character_id INTEGER
		);
		CREATE UNIQUE INDEX idx_characters_metron_character_id
		ON characters(metron_character_id)
		WHERE metron_character_id IS NOT NULL;
		CREATE TABLE user_characters (character_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (character_id, user_id));

		CREATE TABLE character_aliases (
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			PRIMARY KEY (character_id, alias)
		);

		CREATE TABLE comic_characters (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			PRIMARY KEY (comic_id, character_id)
		);

		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_arc_id INTEGER,
			image TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_arcs_metron_arc_id
		ON arcs(metron_arc_id)
		WHERE metron_arc_id IS NOT NULL;
		CREATE TABLE user_arcs (arc_id INTEGER NOT NULL, user_id INTEGER NOT NULL, started_at TEXT, favorite INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (arc_id, user_id));

		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);

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
		CREATE TABLE reading_order_sections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE metron_sync_states (
			resource_type TEXT NOT NULL,
			metron_id INTEGER NOT NULL,
			last_modified TEXT NOT NULL DEFAULT '',
			fully_synced INTEGER NOT NULL DEFAULT 0,
			synced_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (resource_type, metron_id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func serverNextURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}
