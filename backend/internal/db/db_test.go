package db

import (
	"path/filepath"
	"testing"
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
