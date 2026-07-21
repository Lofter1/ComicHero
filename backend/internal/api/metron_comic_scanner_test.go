package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
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
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMetronComicScanSettingsRoundTrip(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	settings := MetronComicScanSettings{Enabled: true, ScanComics: true, Schedule: "weekly", Weekdays: []string{"friday", "monday"}, StartTime: "03:15", DailyCallLimit: 12, MinIntervalSeconds: 20, IncompleteFields: []string{"publisher", "comicVineId"}}
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
	if !got.Enabled || got.DailyCallLimit != 12 || got.MinIntervalSeconds != 20 || len(got.Weekdays) != 2 || len(got.IncompleteFields) != 2 {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestMetronComicScanLegacySettingsKeepDefaultIncompleteFields(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)`, metronComicScanSettingsKey, `{"enabled":true}`); err != nil {
		t.Fatal(err)
	}
	settings, err := loadMetronComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.IncompleteFields) != len(metronComicIncompleteFields) {
		t.Fatalf("legacy incomplete fields = %v; want defaults %v", settings.IncompleteFields, metronComicIncompleteFields)
	}
}

func TestMetronComicScanSettingsRequireKnownIncompleteFields(t *testing.T) {
	settings := defaultMetronComicScanSettings()
	settings.IncompleteFields = nil
	if err := validateMetronComicScanSettings(&settings); err == nil {
		t.Fatal("empty incomplete fields returned nil error")
	}

	settings = defaultMetronComicScanSettings()
	settings.IncompleteFields = append(settings.IncompleteFields, "unknown")
	if err := validateMetronComicScanSettings(&settings); err == nil {
		t.Fatal("unknown incomplete field returned nil error")
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
		comic_vine_id INTEGER,
		metron_synced_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	recentlySynced := now.Add(-1 * time.Hour).Format(time.RFC3339)
	longAgoSynced := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	// Row A: publisher filled, description permanently blank (Metron has none),
	// synced an hour ago -> still "incomplete" but should be skipped by the cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('A', 'Marvel', '1964-01-01', '/covers/a.jpg', '', 1, 101, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}
	// Row B: never synced -> should always be selected.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('B', '', '', '', '', 2, NULL, '')`); err != nil {
		t.Fatal(err)
	}
	// Row C: synced 40 days ago, past the 30-day cooldown -> should be selected again.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('C', 'DC', '1980-01-01', '/covers/c.jpg', '', 3, 103, ?)`, longAgoSynced); err != nil {
		t.Fatal(err)
	}
	// Row D: all legacy metadata is complete, but the Comic Vine ID is missing.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('D', 'Image', '2020-01-01', '/covers/d.jpg', 'Complete', 4, NULL, '')`); err != nil {
		t.Fatal(err)
	}
	// Row E: no Metron ID yet, but its Comic Vine ID can be used to find one.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('E', '', '', '', '', NULL, 105, '')`); err != nil {
		t.Fatal(err)
	}
	// Row F: a recent unsuccessful Comic Vine lookup should respect the cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('F', '', '', '', '', NULL, 106, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}

	settings := defaultMetronComicScanSettings()
	rows, err := selectIncompleteComics(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}

	got := map[int]bool{}
	for _, r := range rows {
		got[r.ID] = true
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
	if !got[4] {
		t.Fatal("comic missing only a Comic Vine ID should be selected")
	}
	if !got[5] {
		t.Fatal("comic with only a Comic Vine ID should be selected")
	}
	if got[6] {
		t.Fatal("recently checked comic with only a Comic Vine ID should be skipped during cooldown")
	}
}

func TestMetronComicScanUsesSelectedIncompleteFields(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY,
			publisher TEXT NOT NULL DEFAULT '',
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			metron_issue_id INTEGER,
			comic_vine_id INTEGER,
			metron_synced_at TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO comics VALUES
			(1, '', '2020-01-01', '/one.jpg', 'Complete', 101, 1001, ''),
			(2, 'Publisher', '2020-01-01', '/two.jpg', 'Complete', 102, NULL, ''),
			(3, 'Publisher', '2020-01-01', '/three.jpg', '', 103, 1003, '');
	`); err != nil {
		t.Fatal(err)
	}

	settings := defaultMetronComicScanSettings()
	settings.RecheckCooldownDays = 0
	settings.IncompleteFields = []string{"publisher"}
	rows, err := selectIncompleteComics(context.Background(), db, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].MetronID.Int64 != 101 {
		t.Fatalf("publisher rows = %+v; want only Metron issue 101", rows)
	}

	settings.IncompleteFields = []string{"comicVineId", "description"}
	rows, err = selectIncompleteComics(context.Background(), db, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].MetronID.Int64 != 102 || rows[1].MetronID.Int64 != 103 {
		t.Fatalf("selected rows = %+v; want Metron issues 102 and 103", rows)
	}
}

func TestMetronComicScanFindsAndCoolsDownComicVineOnlyComics(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO comics (series, issue, publisher, comic_vine_id) VALUES
			('Found', '1', '', 9001),
			('Not found', '2', '', 9002)
	`); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/issue/?cv_id=9001":
			_, _ = w.Write([]byte(`{"results":[{"id":77,"cv_id":9001}]}`))
		case "/issue/77/":
			_, _ = w.Write([]byte(`{"id":77,"cv_id":9001,"number":"1","series":{"name":"Found","year_began":2026,"publisher":{"name":"Publisher"}}}`))
		case "/issue/?cv_id=9002":
			_, _ = w.Write([]byte(`{"results":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.RecheckCooldownDays = 30
	settings.IncompleteFields = []string{"publisher"}
	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	var found struct {
		MetronID  int    `db:"metron_issue_id"`
		Publisher string `db:"publisher"`
	}
	if err := db.Get(&found, `SELECT metron_issue_id, publisher FROM comics WHERE comic_vine_id = 9001`); err != nil {
		t.Fatal(err)
	}
	if found.MetronID != 77 || found.Publisher != "Publisher" {
		t.Fatalf("found comic = %+v; want Metron ID 77 and enriched publisher", found)
	}

	var checkedAt string
	if err := db.Get(&checkedAt, `SELECT metron_synced_at FROM comics WHERE comic_vine_id = 9002`); err != nil {
		t.Fatal(err)
	}
	if checkedAt == "" {
		t.Fatal("unmatched Comic Vine comic was not marked as checked")
	}

	now := time.Now()
	rows, err := selectIncompleteComics(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows during cooldown = %+v; want none", rows)
	}
	rows, err = selectIncompleteComics(context.Background(), db, settings, now.Add(31*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ComicVineID.Int64 != 9002 {
		t.Fatalf("rows after cooldown = %+v; want Comic Vine ID 9002", rows)
	}

	usage := currentMetronComicScanUsage(context.Background(), db, time.Now())
	if usage.Calls != 3 {
		t.Fatalf("Metron calls = %d; want two searches and one detail request", usage.Calls)
	}
}

func TestEnrichIncompleteComicStoresComicVineID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, metron_issue_id)
		VALUES ('Series', 2026, '1', 'Publisher', 77)
	`); err != nil {
		t.Fatalf("seed comic: %v", err)
	}
	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Series:      "Series",
		SeriesYear:  2026,
		Issue:       "1",
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic: %v", err)
	}
	var comicVineID int
	if err := db.Get(&comicVineID, `SELECT comic_vine_id FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read Comic Vine ID: %v", err)
	}
	if comicVineID != 9988 {
		t.Fatalf("Comic Vine ID = %d; want 9988", comicVineID)
	}
}

func TestEnrichIncompleteComicSkipsConflictingComicVineID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Target', 2026, '1', '', 77);
		INSERT INTO comics (id, series, series_year, issue, publisher, comic_vine_id)
		VALUES (2, 'Existing Comic Vine row', 2026, '1', 'Publisher', 9988);
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic with conflicting Comic Vine ID: %v", err)
	}

	var comic struct {
		ComicVineID *int   `db:"comic_vine_id"`
		Publisher   string `db:"publisher"`
	}
	if err := db.Get(&comic, `SELECT comic_vine_id, publisher FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read enriched comic: %v", err)
	}
	if comic.ComicVineID != nil {
		t.Fatalf("conflicting Comic Vine ID was attached: %d", *comic.ComicVineID)
	}
	if comic.Publisher != "Publisher" {
		t.Fatalf("publisher = %q; want metadata enrichment to succeed", comic.Publisher)
	}
}

func TestEnrichIncompleteComicSkipsConflictingMetronID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (id, series, series_year, issue, publisher, comic_vine_id)
		VALUES (1, 'Target', 2026, '1', '', 9988);
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (2, 'Existing Metron row', 2026, '1', 'Publisher', 77);
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic with conflicting Metron ID: %v", err)
	}

	var comic struct {
		MetronID  *int   `db:"metron_issue_id"`
		Publisher string `db:"publisher"`
	}
	if err := db.Get(&comic, `SELECT metron_issue_id, publisher FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read enriched comic: %v", err)
	}
	if comic.MetronID != nil {
		t.Fatalf("conflicting Metron ID was attached: %d", *comic.MetronID)
	}
	if comic.Publisher != "Publisher" {
		t.Fatalf("publisher = %q; want metadata enrichment to succeed", comic.Publisher)
	}
}

func TestMetronComicScanReportsLastFailure(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO comics (id, series, issue, publisher, metron_issue_id)
		VALUES (1, 'Target', '1', '', 77);
	`); err != nil {
		t.Fatalf("seed scanner data: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Metron unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.RecheckCooldownDays = 0
	settings.IncompleteFields = []string{"publisher"}
	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	status := scanner.snapshot(context.Background())
	if status.Failed != 1 {
		t.Fatalf("failed = %d; want 1", status.Failed)
	}
	if !strings.Contains(status.LastError, "comic 1: fetch Metron issue") {
		t.Fatalf("last error = %q; want comic and failure stage", status.LastError)
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
