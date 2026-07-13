package api

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newMetronComicScannerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMetronComicScanSettingsRoundTrip(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	settings := MetronComicScanSettings{Enabled: true, ScanComics: true, Schedule: "weekly", Weekdays: []string{"friday", "monday"}, StartTime: "03:15", DailyCallLimit: 12, MinIntervalSeconds: 20}
	if err := validateMetronComicScanSettings(&settings); err != nil {
		t.Fatal(err)
	}
	if err := saveMetronComicScanSettings(context.Background(), db, settings); err != nil {
		t.Fatal(err)
	}
	got, err := loadMetronComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.DailyCallLimit != 12 || got.MinIntervalSeconds != 20 || len(got.Weekdays) != 2 {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestMetronComicScanDailyQuotaIsSharedAndResets(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	ctx := context.Background()
	dayOne := time.Date(2026, 7, 10, 10, 0, 0, 0, time.Local)
	for i := range 2 {
		claimed, err := claimMetronComicScanCall(ctx, db, 2, dayOne)
		if err != nil || !claimed {
			t.Fatalf("claim %d: claimed=%v err=%v", i, claimed, err)
		}
	}
	claimed, err := claimMetronComicScanCall(ctx, db, 2, dayOne)
	if err != nil || claimed {
		t.Fatalf("quota should be exhausted: claimed=%v err=%v", claimed, err)
	}
	claimed, err = claimMetronComicScanCall(ctx, db, 2, dayOne.Add(24*time.Hour))
	if err != nil || !claimed {
		t.Fatalf("quota should reset: claimed=%v err=%v", claimed, err)
	}
}

func TestMetronComicScanSubscriptionSendsSnapshotAndProgress(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	scanner := NewMetronComicScanner(db, nil, nil)
	updates, unsubscribe := scanner.subscribe(context.Background())
	defer unsubscribe()

	initial := <-updates
	if initial.Scanned != 0 {
		t.Fatalf("initial scanned = %d", initial.Scanned)
	}
	scanner.setScanned(3)
	progress := <-updates
	if progress.Scanned != 3 {
		t.Fatalf("progress scanned = %d", progress.Scanned)
	}
}

func TestMetronComicScanCooldownExcludesRecentlySyncedComics(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`CREATE TABLE comics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		series TEXT NOT NULL DEFAULT '',
		series_year INTEGER NOT NULL DEFAULT 0,
		issue TEXT NOT NULL DEFAULT '',
		publisher TEXT NOT NULL DEFAULT '',
		cover_date TEXT NOT NULL DEFAULT '',
		cover_image TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		read INTEGER NOT NULL DEFAULT 0,
		metron_issue_id INTEGER,
		metron_synced_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	recentlySynced := now.Add(-1 * time.Hour).Format(time.RFC3339)
	longAgoSynced := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	// Row A: publisher filled, description permanently blank (Metron has none),
	// synced an hour ago -> still "incomplete" but should be skipped by the cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, metron_synced_at)
		VALUES ('A', 'Marvel', '1964-01-01', '/covers/a.jpg', '', 1, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}
	// Row B: never synced -> should always be selected.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, metron_synced_at)
		VALUES ('B', '', '', '', '', 2, '')`); err != nil {
		t.Fatal(err)
	}
	// Row C: synced 40 days ago, past the 30-day cooldown -> should be selected again.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, metron_synced_at)
		VALUES ('C', 'DC', '1980-01-01', '/covers/c.jpg', '', 3, ?)`, longAgoSynced); err != nil {
		t.Fatal(err)
	}

	cutoff := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	var rows []struct {
		ID       int `db:"id"`
		MetronID int `db:"metron_issue_id"`
	}
	query := `SELECT id, metron_issue_id FROM comics WHERE metron_issue_id IS NOT NULL AND (TRIM(publisher) = '' OR TRIM(cover_image) = '' OR TRIM(cover_date) = '' OR TRIM(description) = '') AND (metron_synced_at = '' OR metron_synced_at <= ?) ORDER BY id`
	if err := db.Select(&rows, query, cutoff); err != nil {
		t.Fatal(err)
	}

	got := map[int]bool{}
	for _, r := range rows {
		got[r.MetronID] = true
	}
	if got[1] {
		t.Fatal("recently-synced comic with a permanently-blank field should be skipped during cooldown")
	}
	if !got[2] {
		t.Fatal("never-synced comic should always be selected")
	}
	if !got[3] {
		t.Fatal("comic synced past the cooldown window should be selected again")
	}
}

func TestComicScanIntervalIsScopedToOneRun(t *testing.T) {
	var firstRunNext time.Time
	if err := waitForComicScanInterval(context.Background(), &firstRunNext, time.Second); err != nil {
		t.Fatal(err)
	}
	if firstRunNext.IsZero() {
		t.Fatal("first scan run did not record its next request time")
	}

	var otherRunNext time.Time
	started := time.Now()
	if err := waitForComicScanInterval(context.Background(), &otherRunNext, time.Second); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("independent scan run waited %v for another run", elapsed)
	}
}
