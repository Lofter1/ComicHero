package api

import "testing"

func TestMergeComicPreservesRelationshipsAndFillsMissingMetadata(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		UPDATE users SET is_admin = 1 WHERE id = 1;
		INSERT INTO users (id, name) VALUES (2, 'Reader');
		INSERT INTO comics (
			id, series, series_year, issue, publisher, cover_date, cover_image, description,
			metron_issue_id, comic_vine_id, metron_synced_at
		) VALUES
			(1, 'Target Series', 2020, '1', '', '', '', '', NULL, NULL, ''),
			(2, 'Source Series', 2021, '2', 'Publisher', '2021-02-01', '/cover.jpg',
			 'Source description', 2002, 4002, '2026-01-02T00:00:00Z');
		INSERT INTO reading_orders (id, name) VALUES (1, 'Order');
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position, note)
		VALUES (1, 1, 1, 'target position'), (1, 2, 2, 'source position');
		INSERT INTO arcs (id, name) VALUES (1, 'Arc');
		INSERT INTO arc_comics (arc_id, comic_id, position, note)
		VALUES (1, 1, 1, 'target position'), (1, 2, 2, 'source position');
		INSERT INTO characters (id, name) VALUES (1, 'Shared'), (2, 'Source only');
		INSERT INTO comic_characters (comic_id, character_id)
		VALUES (1, 1), (2, 1), (2, 2);
		INSERT INTO user_comics (comic_id, user_id, read, skipped, read_at)
		VALUES
			(1, 1, 0, 1, ''),
			(2, 1, 1, 0, '2026-01-02T00:00:00Z'),
			(2, 2, 1, 0, '2026-01-01T00:00:00Z');
	`); err != nil {
		t.Fatal(err)
	}

	output, err := mergeComic(deletionUserContext(1), db, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if output.Body.ID != 1 {
		t.Fatalf("merged comic id = %d, want 1", output.Body.ID)
	}
	if output.Body.Publisher != "Publisher" || output.Body.Description != "Source description" {
		t.Fatalf("missing metadata was not filled: %+v", output.Body.Comic)
	}
	if output.Body.Series != "Target Series" || output.Body.Issue != "1" {
		t.Fatalf("target metadata was overwritten: %+v", output.Body.Comic)
	}
	if output.Body.MetronIssueID == nil || *output.Body.MetronIssueID != 2002 {
		t.Fatalf("metron issue id = %v, want 2002", output.Body.MetronIssueID)
	}
	if output.Body.ComicVineID == nil || *output.Body.ComicVineID != 4002 {
		t.Fatalf("comic vine id = %v, want 4002", output.Body.ComicVineID)
	}

	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM comics WHERE id = 2`, 0)
	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM reading_order_comics WHERE comic_id = 1`, 2)
	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM arc_comics WHERE comic_id = 1`, 2)
	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM comic_characters WHERE comic_id = 1`, 2)
	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM user_comics WHERE comic_id = 1`, 2)

	var state struct {
		Read    bool   `db:"read"`
		Skipped bool   `db:"skipped"`
		ReadAt  string `db:"read_at"`
	}
	if err := db.Get(&state, `SELECT read, skipped, read_at FROM user_comics WHERE comic_id = 1 AND user_id = 1`); err != nil {
		t.Fatal(err)
	}
	if !state.Read || !state.Skipped || state.ReadAt != "2026-01-02T00:00:00Z" {
		t.Fatalf("merged user state = %+v", state)
	}
}

func TestMergeComicRequiresAdminAndDistinctExistingComics(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO users (id, name) VALUES (2, 'Reader');
		INSERT INTO comics (id, series, issue, publisher) VALUES
			(1, 'Series', '1', 'Publisher'),
			(2, 'Series', '1', 'Publisher');
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := mergeComic(deletionUserContext(2), db, 1, 2); err == nil {
		t.Fatal("non-admin merge returned nil error")
	}
	if _, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := mergeComic(deletionUserContext(1), db, 1, 1); err == nil {
		t.Fatal("self merge returned nil error")
	}
	if _, err := mergeComic(deletionUserContext(1), db, 1, 999); err == nil {
		t.Fatal("missing source merge returned nil error")
	}
}

func assertComicMergeCount(t *testing.T, db interface {
	Get(dest any, query string, args ...any) error
}, query string, want int) {
	t.Helper()
	var got int
	if err := db.Get(&got, query); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("query %q count = %d, want %d", query, got, want)
	}
}
