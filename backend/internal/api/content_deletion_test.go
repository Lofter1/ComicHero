package api

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
)

func newContentDeletionTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:?_fk=true")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, is_admin INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT NOT NULL, series_year INTEGER NOT NULL DEFAULT 0);
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY,
			series TEXT NOT NULL DEFAULT '',
			series_year INTEGER NOT NULL DEFAULT 0,
			series_id INTEGER REFERENCES series(id) ON DELETE SET NULL
		);
		CREATE TABLE arcs (id INTEGER PRIMARY KEY);
		CREATE TABLE characters (id INTEGER PRIMARY KEY);
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		INSERT INTO users (id, is_admin) VALUES (1, 1), (2, 0), (3, 0);
		INSERT INTO series (id, name, series_year) VALUES (1, 'Series', 2020);
		INSERT INTO comics (id, series, series_year, series_id) VALUES (1, 'Series', 2020, 1);
		INSERT INTO arcs (id) VALUES (1);
		INSERT INTO characters (id) VALUES (1);
		INSERT INTO reading_orders (id, author_user_id) VALUES (1, 2), (2, 3);
	`); err != nil {
		t.Fatal(err)
	}
	return db
}

func deletionUserContext(userID int) context.Context {
	return context.WithValue(context.Background(), contextUserIDKey{}, userID)
}

func TestContentDeletionRequiresAdmin(t *testing.T) {
	for name, deleteContent := range map[string]func(context.Context, *sqlx.DB) error{
		"comic":     func(ctx context.Context, db *sqlx.DB) error { _, err := deleteComic(ctx, db, 1); return err },
		"arc":       func(ctx context.Context, db *sqlx.DB) error { _, err := deleteArc(ctx, db, 1); return err },
		"character": func(ctx context.Context, db *sqlx.DB) error { _, err := deleteCharacter(ctx, db, 1); return err },
		"series":    func(ctx context.Context, db *sqlx.DB) error { _, err := deleteSeries(ctx, db, 1); return err },
	} {
		t.Run(name, func(t *testing.T) {
			db := newContentDeletionTestDB(t)
			if err := deleteContent(deletionUserContext(2), db); err == nil {
				t.Fatal("non-admin deletion returned nil error")
			}
			if err := deleteContent(deletionUserContext(1), db); err != nil {
				t.Fatalf("admin deletion failed: %v", err)
			}
		})
	}
}

func TestReadingOrderDeletionAllowsAuthorOrAdmin(t *testing.T) {
	db := newContentDeletionTestDB(t)
	if _, err := deleteReadingOrder(deletionUserContext(2), db, 2); err == nil {
		t.Fatal("non-author deletion returned nil error")
	}
	if _, err := deleteReadingOrder(deletionUserContext(2), db, 1); err != nil {
		t.Fatalf("author deletion failed: %v", err)
	}
	if _, err := deleteReadingOrder(deletionUserContext(1), db, 2); err != nil {
		t.Fatalf("admin deletion failed: %v", err)
	}
}

func TestSeriesDeletionAlsoDeletesLinkedComics(t *testing.T) {
	db := newContentDeletionTestDB(t)
	if _, err := deleteSeries(deletionUserContext(1), db, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("linked comic count = %d, want 0", count)
	}
}
