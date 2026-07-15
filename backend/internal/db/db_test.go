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
	if !columns["series_id"] {
		t.Fatal("comics table missing series_id column")
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
		"idx_comics_series_id_issue",
		"idx_comics_series_year_publisher",
		"idx_comics_series_year_cover",
	} {
		if !indexes[name] {
			t.Fatalf("comics table missing index %s", name)
		}
	}

	rows, err = database.Query(`PRAGMA table_info(reading_orders)`)
	if err != nil {
		t.Fatalf("reading order columns: %v", err)
	}
	defer rows.Close()

	readingOrderColumns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan reading order column: %v", err)
		}
		readingOrderColumns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading order columns: %v", err)
	}
	for _, name := range []string{"rating", "rating_count", "is_public"} {
		if !readingOrderColumns[name] {
			t.Fatalf("reading_orders.%s missing", name)
		}
	}

	var ratingIndexCount int
	if err := database.Get(&ratingIndexCount, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'idx_reading_orders_rating'
	`); err != nil {
		t.Fatalf("check reading order rating index: %v", err)
	}
	if ratingIndexCount != 1 {
		t.Fatalf("idx_reading_orders_rating count = %d; want 1", ratingIndexCount)
	}

	var userRatingTableCount int
	if err := database.Get(&userRatingTableCount, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'reading_order_ratings'
	`); err != nil {
		t.Fatalf("check reading order user rating table: %v", err)
	}
	if userRatingTableCount != 1 {
		t.Fatalf("reading_order_ratings table count = %d; want 1", userRatingTableCount)
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

	for _, table := range []string{"app_settings", "user_sessions", "user_comics", "user_email_verifications", "user_password_resets"} {
		exists, err := tableExists(database, table)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("table %s missing", table)
		}
	}
	for _, column := range []string{"email", "email_verified_at", "password_hash", "is_default", "created_at"} {
		exists, err := columnExists(database, "users", column)
		if err != nil {
			t.Fatalf("check column %s: %v", column, err)
		}
		if !exists {
			t.Fatalf("users.%s missing", column)
		}
	}
	sessionExpiryExists, err := columnExists(database, "user_sessions", "expires_at")
	if err != nil {
		t.Fatalf("check user_sessions.expires_at: %v", err)
	}
	if !sessionExpiryExists {
		t.Fatal("user_sessions.expires_at missing")
	}
	var emailIndexCount int
	if err := database.Get(&emailIndexCount, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_users_email'`); err != nil {
		t.Fatalf("check email index: %v", err)
	}
	if emailIndexCount != 1 {
		t.Fatalf("idx_users_email count = %d; want 1", emailIndexCount)
	}
	if _, err := database.Exec(`
		INSERT INTO users (id, name, email, email_verified_at)
		VALUES (2, 'Pending', 'pending@example.com', '');
		INSERT INTO user_email_verifications (token_hash, user_id, expires_at)
		VALUES ('pending-token-hash', 2, '2999-01-01T00:00:00Z');
	`); err != nil {
		t.Fatalf("seed pending verification user: %v", err)
	}
	if err := ensureUserLoginSchema(database); err != nil {
		t.Fatalf("ensure user login schema after pending user: %v", err)
	}
	var pendingVerifiedAt string
	if err := database.Get(&pendingVerifiedAt, `SELECT email_verified_at FROM users WHERE id = 2`); err != nil {
		t.Fatalf("fetch pending email verification: %v", err)
	}
	if pendingVerifiedAt != "" {
		t.Fatalf("pending email_verified_at = %q; want empty", pendingVerifiedAt)
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
	skippedExists, err := columnExists(database, "user_comics", "skipped")
	if err != nil {
		t.Fatalf("check user_comics.skipped: %v", err)
	}
	if !skippedExists {
		t.Fatal("user_comics.skipped missing")
	}
	var skipped int
	if err := database.Get(&skipped, `
		SELECT uc.skipped
		FROM user_comics uc
		JOIN users u ON u.id = uc.user_id
		JOIN comics c ON c.id = uc.comic_id
		WHERE u.name = 'Default' AND c.series = 'Series'
	`); err != nil {
		t.Fatalf("fetch backfilled skipped status: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("backfilled skipped = %d; want 0", skipped)
	}
}

func TestEnsureUserLoginSchemaAddsSeriesMetronIDToLegacySeries(t *testing.T) {
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
		CREATE TABLE series (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0
		);
		CREATE UNIQUE INDEX idx_series_name_year
		ON series(name, series_year);
		CREATE TABLE comics (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			series      TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue       TEXT NOT NULL,
			publisher   TEXT NOT NULL
		);
		INSERT INTO users (name) VALUES ('Default');
		INSERT INTO series (name, series_year) VALUES ('Legacy Series', 2026);
		INSERT INTO comics (series, series_year, issue, publisher) VALUES ('Legacy Series', 2026, '1', 'Publisher');
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	if err := ensureUserLoginSchema(database); err != nil {
		t.Fatalf("ensure user login schema: %v", err)
	}

	hasMetronSeriesID, err := columnExists(database, "series", "metron_series_id")
	if err != nil {
		t.Fatalf("check series.metron_series_id: %v", err)
	}
	if !hasMetronSeriesID {
		t.Fatal("series.metron_series_id missing")
	}

	var indexCount int
	if err := database.Get(&indexCount, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_series_metron_series_id'`); err != nil {
		t.Fatalf("check series metron index: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("idx_series_metron_series_id count = %d; want 1", indexCount)
	}
}
