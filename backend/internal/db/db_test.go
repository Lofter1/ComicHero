package db

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestOpenAppliesComicGeneratedTitleMigration(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "comicorder.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	rows, err := database.Query(`PRAGMA table_info(comics)`)
	if err != nil {
		t.Fatalf("table info: %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("columns: %v", err)
	}

	if columns["title"] {
		t.Fatal("comics table still has title column")
	}
	if !columns["series_year"] {
		t.Fatal("comics table missing series_year column")
	}
	var busyTimeout int
	if err := database.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("busy timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy timeout = %d; want 5000", busyTimeout)
	}

	var seriesTable string
	if err := database.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = 'series'
	`).Scan(&seriesTable); err != nil {
		t.Fatal("series table missing")
	}

	rows, err = database.Query(`PRAGMA index_list(comics)`)
	if err != nil {
		t.Fatalf("comic indexes: %v", err)
	}
	defer rows.Close()

	indexes := map[string]bool{}
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index: %v", err)
		}
		indexes[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("indexes: %v", err)
	}
	for _, name := range []string{
		"idx_comics_series_year_issue",
		"idx_comics_series_year_publisher",
		"idx_comics_series_year_cover",
	} {
		if !indexes[name] {
			t.Fatalf("comics table missing index %s", name)
		}
	}
}

func TestEnsureUserLoginSchemaUpgradesMergedMigrationDrift(t *testing.T) {
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`
		CREATE TABLE users (
			id   INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);
		CREATE TABLE comics (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			series    TEXT NOT NULL,
			issue     TEXT NOT NULL,
			publisher TEXT NOT NULL,
			read      INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO users (name) VALUES ('Default');
		INSERT INTO comics (series, issue, publisher, read) VALUES ('Series', '1', 'Publisher', 1);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	if err := ensureUserLoginSchema(database); err != nil {
		t.Fatalf("ensure user login schema: %v", err)
	}

	for _, table := range []string{"app_settings", "user_sessions", "user_comics"} {
		exists, err := tableExists(database, table)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("table %s missing", table)
		}
	}
	for _, column := range []string{"password_hash", "is_default", "created_at"} {
		exists, err := columnExists(database, "users", column)
		if err != nil {
			t.Fatalf("check column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("users.%s missing", column)
		}
	}

	var isDefault int
	if err := database.Get(&isDefault, `SELECT is_default FROM users WHERE name = 'Default'`); err != nil {
		t.Fatalf("fetch default user: %v", err)
	}
	if isDefault != 1 {
		t.Fatalf("is_default = %d; want 1", isDefault)
	}

	var read int
	if err := database.Get(&read, `
		SELECT uc.read
		FROM user_comics uc
		JOIN users u ON u.id = uc.user_id
		JOIN comics c ON c.id = uc.comic_id
		WHERE u.name = 'Default' AND c.series = 'Series'
	`); err != nil {
		t.Fatalf("fetch backfilled read status: %v", err)
	}
	if read != 1 {
		t.Fatalf("backfilled read = %d; want 1", read)
	}
}
