package api

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestContentPreferencesArePerUserAndPreserveIndependentState(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY);
		CREATE TABLE series (id INTEGER PRIMARY KEY);
		CREATE TABLE user_series (
			series_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			started_at TEXT,
			favorite INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (series_id, user_id)
		);
		INSERT INTO users (id) VALUES (1), (2);
		INSERT INTO series (id) VALUES (10);
	`); err != nil {
		t.Fatal(err)
	}

	first := context.WithValue(context.Background(), contextUserIDKey{}, 1)
	second := context.WithValue(context.Background(), contextUserIDKey{}, 2)
	if err := setContentFavorite(first, db, "user_series", "series_id", "series", 10, true); err != nil {
		t.Fatal(err)
	}
	if err := setContentStarted(first, db, "user_series", "series_id", "series", 10, true); err != nil {
		t.Fatal(err)
	}
	if err := setContentStarted(second, db, "user_series", "series_id", "series", 10, true); err != nil {
		t.Fatal(err)
	}
	if err := setContentStarted(first, db, "user_series", "series_id", "series", 10, false); err != nil {
		t.Fatal(err)
	}

	var firstFavorite bool
	var firstStarted *string
	if err := db.QueryRowx(`SELECT favorite, started_at FROM user_series WHERE series_id = 10 AND user_id = 1`).Scan(&firstFavorite, &firstStarted); err != nil {
		t.Fatal(err)
	}
	if !firstFavorite || firstStarted != nil {
		t.Fatalf("first user state = favorite %v, started %v; want true, nil", firstFavorite, firstStarted)
	}

	var secondFavorite bool
	var secondStarted *string
	if err := db.QueryRowx(`SELECT favorite, started_at FROM user_series WHERE series_id = 10 AND user_id = 2`).Scan(&secondFavorite, &secondStarted); err != nil {
		t.Fatal(err)
	}
	if secondFavorite || secondStarted == nil {
		t.Fatalf("second user state = favorite %v, started %v; want false, non-nil", secondFavorite, secondStarted)
	}
}
