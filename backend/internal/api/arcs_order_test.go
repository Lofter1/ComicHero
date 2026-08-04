package api

import (
	"context"
	"testing"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestArcIssuePosition(t *testing.T) {
	issues := []metron.Issue{
		{ID: 10},
		{ID: 20},
		{ID: 30},
	}

	if got := arcIssuePosition(issues, 20); got != 2 {
		t.Fatalf("position for issue 20 = %d, want 2", got)
	}
	if got := arcIssuePosition(issues, 10); got != 1 {
		t.Fatalf("position for issue 10 = %d, want 1", got)
	}
	if got := arcIssuePosition(issues, 999); got != 0 {
		t.Fatalf("position for unknown issue = %d, want 0 (unset)", got)
	}
	if got := arcIssuePosition(issues, 0); got != 0 {
		t.Fatalf("position for zero issue ID = %d, want 0", got)
	}
	if got := arcIssuePosition(nil, 10); got != 0 {
		t.Fatalf("position with no issues = %d, want 0", got)
	}
}

func TestApplyArcIssueOrderUpdatesOnlyLocallyOwnedComics(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_issue_id INTEGER
		);
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL,
			comic_id INTEGER NOT NULL,
			position INTEGER NOT NULL DEFAULT 0
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Comic 1 (metron issue 10) and comic 2 (metron issue 30) are already
	// linked to arc 1 with unset positions; issue 20 isn't owned locally.
	if _, err := db.Exec(`INSERT INTO comics (id, metron_issue_id) VALUES (1, 10), (2, 30)`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO arc_comics (arc_id, comic_id, position) VALUES (1, 1, 0), (1, 2, 0)`); err != nil {
		t.Fatalf("seed arc_comics: %v", err)
	}

	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	issues := []metron.Issue{{ID: 10}, {ID: 20}, {ID: 30}}
	if err := applyArcIssueOrder(context.Background(), tx, 1, issues); err != nil {
		t.Fatalf("applyArcIssueOrder: %v", err)
	}

	positions := map[int]int{}
	rows, err := tx.QueryContext(context.Background(), `SELECT comic_id, position FROM arc_comics WHERE arc_id = 1`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var comicID, position int
		if err := rows.Scan(&comicID, &position); err != nil {
			t.Fatalf("scan: %v", err)
		}
		positions[comicID] = position
	}

	if positions[1] != 1 {
		t.Fatalf("comic 1 (issue 10) position = %d, want 1", positions[1])
	}
	if positions[2] != 3 {
		t.Fatalf("comic 2 (issue 30) position = %d, want 3", positions[2])
	}
}
