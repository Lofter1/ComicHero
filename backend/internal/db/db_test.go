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

	var seriesTable string
	if err := database.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name = 'series'
	`).Scan(&seriesTable); err != nil {
		t.Fatal("series table missing")
	}
}
