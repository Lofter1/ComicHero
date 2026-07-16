package api

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

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
		created_at TEXT NOT NULL DEFAULT '',
		last_login_at TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			is_public INTEGER NOT NULL DEFAULT 1,
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
