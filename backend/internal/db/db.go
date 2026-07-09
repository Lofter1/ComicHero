package db

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(path string) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sqlx.Open("sqlite", path+"?_fk=true&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := configureSQLite(db); err != nil {
		return nil, fmt.Errorf("configure sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	if err := ensureUserLoginSchema(db); err != nil {
		return nil, fmt.Errorf("user login schema: %w", err)
	}

	return db, nil
}

func configureSQLite(db *sqlx.DB) error {
	pragmas := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return err
		}
	}
	return nil
}

func runMigrations(db *sqlx.DB) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	return goose.Up(db.DB, "migrations")
}

func ensureUserLoginSchema(db *sqlx.DB) error {
	exists, err := tableExists(db, "users")
	if err != nil || !exists {
		return err
	}

	columns := []struct {
		name string
		sql  string
	}{
		{name: "email", sql: `ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT ''`},
		{name: "password_hash", sql: `ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`},
		{name: "is_default", sql: `ALTER TABLE users ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0`},
		{name: "is_admin", sql: `ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0`},
		{name: "created_at", sql: `ALTER TABLE users ADD COLUMN created_at TEXT NOT NULL DEFAULT ''`},
		{name: "email_verified_at", sql: `ALTER TABLE users ADD COLUMN email_verified_at TEXT NOT NULL DEFAULT ''`},
	}
	for _, column := range columns {
		exists, err := columnExists(db, "users", column.name)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.Exec(column.sql); err != nil {
				return err
			}
		}
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS app_settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_sessions (
			token      TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	hasSessionExpiry, err := columnExists(db, "user_sessions", "expires_at")
	if err != nil {
		return err
	}
	if !hasSessionExpiry {
		if _, err := db.Exec(`ALTER TABLE user_sessions ADD COLUMN expires_at TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read     INTEGER NOT NULL DEFAULT 0,
			skipped  INTEGER NOT NULL DEFAULT 0,
			read_at  TEXT    NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		)
	`); err != nil {
		return err
	}
	hasReadAt, err := columnExists(db, "user_comics", "read_at")
	if err != nil {
		return err
	}
	if !hasReadAt {
		if _, err := db.Exec(`ALTER TABLE user_comics ADD COLUMN read_at TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`UPDATE user_comics SET read_at = CURRENT_TIMESTAMP WHERE read = 1 AND read_at = ''`); err != nil {
		return err
	}
	hasSkipped, err := columnExists(db, "user_comics", "skipped")
	if err != nil {
		return err
	}
	if !hasSkipped {
		if _, err := db.Exec(`ALTER TABLE user_comics ADD COLUMN skipped INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_user_comics_user_skipped ON user_comics(user_id, skipped)`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE users SET is_default = 1 WHERE name = 'Default' AND is_default = 0`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email <> ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO users (name, is_default) VALUES ('Default', 1)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE users
		SET is_admin = 1
		WHERE is_default = 1
		   OR id = (SELECT MIN(id) FROM users)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE users SET email_verified_at = CURRENT_TIMESTAMP WHERE email_verified_at = '' AND (is_default = 1 OR is_admin = 1)`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_metron_permissions (
			user_id      INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			allowed      INTEGER NOT NULL DEFAULT 0,
			scopes       TEXT    NOT NULL DEFAULT '',
			hourly_limit INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at   TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_metron_request_log (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			scope      TEXT    NOT NULL,
			endpoint   TEXT    NOT NULL,
			created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_metron_request_log_user_created
		ON user_metron_request_log(user_id, created_at)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_email_verifications (
			token_hash TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT    NOT NULL,
			used_at    TEXT    NOT NULL DEFAULT '',
			created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_email_verifications_user_expires
		ON user_email_verifications(user_id, expires_at)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_password_resets (
			token_hash TEXT PRIMARY KEY,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT    NOT NULL,
			used_at    TEXT    NOT NULL DEFAULT '',
			created_at TEXT    NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_password_resets_user_expires
		ON user_password_resets(user_id, expires_at)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE users
		SET email_verified_at = ''
		WHERE id IN (
			SELECT user_id
			FROM user_email_verifications
			WHERE used_at = ''
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit)
		SELECT id, 1, '*', 0
		FROM users
		WHERE is_admin = 1
		ON CONFLICT(user_id) DO NOTHING
	`); err != nil {
		return err
	}
	if err := ensureReadingOrderAuthors(db); err != nil {
		return err
	}
	if err := ensureReadingOrderRatings(db); err != nil {
		return err
	}
	if err := ensureSeriesMetronSchema(db); err != nil {
		return err
	}
	if err := ensureComicSeriesLinks(db); err != nil {
		return err
	}

	hasComicRead, err := columnExists(db, "comics", "read")
	if err != nil {
		return err
	}
	if hasComicRead {
		_, err = db.Exec(`
			INSERT OR IGNORE INTO user_comics (comic_id, user_id, read, skipped, read_at)
			SELECT c.id, u.id, c.read, 0, CASE WHEN c.read = 1 THEN CURRENT_TIMESTAMP ELSE '' END
			FROM comics c
			JOIN users u ON u.is_default = 1
		`)
		return err
	}
	return err
}

func ensureReadingOrderAuthors(db *sqlx.DB) error {
	exists, err := tableExists(db, "reading_orders")
	if err != nil || !exists {
		return err
	}

	hasAuthor, err := columnExists(db, "reading_orders", "author_user_id")
	if err != nil {
		return err
	}
	if !hasAuthor {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		UPDATE reading_orders
		SET author_user_id = (
			SELECT id
			FROM users
			WHERE is_default = 1 OR name = 'Default'
			ORDER BY is_default DESC, id
			LIMIT 1
		)
		WHERE author_user_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reading_orders_author_user_id
		ON reading_orders(author_user_id)
	`); err != nil {
		return err
	}
	return nil
}

func ensureReadingOrderRatings(db *sqlx.DB) error {
	exists, err := tableExists(db, "reading_orders")
	if err != nil || !exists {
		return err
	}

	hasRating, err := columnExists(db, "reading_orders", "rating")
	if err != nil {
		return err
	}
	if !hasRating {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN rating REAL NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	hasRatingCount, err := columnExists(db, "reading_orders", "rating_count")
	if err != nil {
		return err
	}
	if !hasRatingCount {
		if _, err := db.Exec(`ALTER TABLE reading_orders ADD COLUMN rating_count INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reading_orders_rating
		ON reading_orders(rating)
	`); err != nil {
		return err
	}
	return nil
}

func ensureSeriesMetronSchema(db *sqlx.DB) error {
	exists, err := tableExists(db, "series")
	if err != nil || !exists {
		return err
	}

	hasMetronSeriesID, err := columnExists(db, "series", "metron_series_id")
	if err != nil {
		return err
	}
	if !hasMetronSeriesID {
		if _, err := db.Exec(`ALTER TABLE series ADD COLUMN metron_series_id INTEGER`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_series_metron_series_id
		ON series(metron_series_id)
		WHERE metron_series_id IS NOT NULL
	`); err != nil {
		return err
	}
	return nil
}

func ensureComicSeriesLinks(db *sqlx.DB) error {
	comicsExists, err := tableExists(db, "comics")
	if err != nil || !comicsExists {
		return err
	}
	seriesExists, err := tableExists(db, "series")
	if err != nil || !seriesExists {
		return err
	}

	hasSeriesID, err := columnExists(db, "comics", "series_id")
	if err != nil {
		return err
	}
	if !hasSeriesID {
		if _, err := db.Exec(`ALTER TABLE comics ADD COLUMN series_id INTEGER REFERENCES series(id) ON DELETE SET NULL`); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO series (name, series_year)
		SELECT DISTINCT series, series_year
		FROM comics
		WHERE TRIM(series) <> ''
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		UPDATE comics
		SET series_id = (
			SELECT id
			FROM series
			WHERE series.name = comics.series
			  AND series.series_year = comics.series_year
			LIMIT 1
		)
		WHERE series_id IS NULL
		  AND TRIM(series) <> ''
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_comics_series_id_issue
		ON comics(series_id, issue)
	`); err != nil {
		return err
	}
	return nil
}

func tableExists(db *sqlx.DB, name string) (bool, error) {
	var count int
	if err := db.Get(&count, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = ?
	`, name); err != nil {
		return false, err
	}
	return count > 0, nil
}

func columnExists(db *sqlx.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
