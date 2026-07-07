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
		{name: "password_hash", sql: `ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`},
		{name: "is_default", sql: `ALTER TABLE users ADD COLUMN is_default INTEGER NOT NULL DEFAULT 0`},
		{name: "is_admin", sql: `ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0`},
		{name: "created_at", sql: `ALTER TABLE users ADD COLUMN created_at TEXT NOT NULL DEFAULT ''`},
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
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read     INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (comic_id, user_id)
		)
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE users SET is_default = 1 WHERE name = 'Default' AND is_default = 0`); err != nil {
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
		INSERT INTO user_metron_permissions (user_id, allowed, scopes, hourly_limit)
		SELECT id, 1, '*', 0
		FROM users
		WHERE is_admin = 1
		ON CONFLICT(user_id) DO NOTHING
	`); err != nil {
		return err
	}

	hasComicRead, err := columnExists(db, "comics", "read")
	if err != nil {
		return err
	}
	if hasComicRead {
		_, err = db.Exec(`
			INSERT OR IGNORE INTO user_comics (comic_id, user_id, read)
			SELECT c.id, u.id, c.read
			FROM comics c
			JOIN users u ON u.is_default = 1
		`)
		return err
	}
	return err
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
