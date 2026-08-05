package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Lofter1/ComicHero/backend/comicvine"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newComicVineComicScannerTestDB(t *testing.T) *sqlx.DB {
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
		metron_synced_at TEXT NOT NULL DEFAULT '',
		comic_vine_synced_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestComicVineComicScanSettingsRoundTrip(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	settings := ComicVineComicScanSettings{
		Enabled: true, Schedule: "weekly", Weekdays: []string{"friday", "monday"},
		StartTime: "03:15", DailyCallLimit: 12, MinIntervalSeconds: 20, RecheckCooldownDays: 14,
		IncompleteFields: []string{"publisher", "description"},
	}
	if err := validateComicVineComicScanSettings(&settings); err != nil {
		t.Fatal(err)
	}
	if err := saveComicVineComicScanSettings(context.Background(), db, settings); err != nil {
		t.Fatal(err)
	}
	got, err := loadComicVineComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.DailyCallLimit != 12 || got.MinIntervalSeconds != 20 ||
		got.RecheckCooldownDays != 14 || len(got.Weekdays) != 2 || len(got.IncompleteFields) != 2 {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestComicVineComicScanLegacySettingsKeepDefaultIncompleteFields(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)`, comicVineComicScanSettingsKey, `{"enabled":true}`); err != nil {
		t.Fatal(err)
	}
	settings, err := loadComicVineComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.IncompleteFields) != len(comicVineComicIncompleteFields) {
		t.Fatalf("legacy incomplete fields = %v; want defaults %v", settings.IncompleteFields, comicVineComicIncompleteFields)
	}
}

func TestValidateComicVineComicScanSettingsRejectsUnknownField(t *testing.T) {
	settings := defaultComicVineComicScanSettings()
	settings.IncompleteFields = []string{"comicVineId"}
	if err := validateComicVineComicScanSettings(&settings); err == nil {
		t.Fatal("expected an error for an unknown incomplete field")
	}
}

func TestValidateComicVineComicScanSettingsRequiresWeekdaysForWeeklySchedule(t *testing.T) {
	settings := defaultComicVineComicScanSettings()
	settings.Schedule = "weekly"
	settings.Weekdays = nil
	if err := validateComicVineComicScanSettings(&settings); err == nil {
		t.Fatal("expected an error for a weekly schedule with no weekdays")
	}
}

func TestComicVineComicScanCooldownExcludesRecentlySyncedComics(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	now := time.Now().UTC()
	recentlySynced := now.Add(-1 * time.Hour).Format(time.RFC3339)
	longAgoSynced := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	// Row A: no Comic Vine ID at all -> never eligible for Comic Vine maintenance.
	if _, err := db.Exec(`INSERT INTO comics (series, comic_vine_id, comic_vine_synced_at) VALUES ('A', NULL, '')`); err != nil {
		t.Fatal(err)
	}
	// Row B: has a Comic Vine ID, missing fields, synced recently -> skipped by cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, comic_vine_id, comic_vine_synced_at)
		VALUES ('B', 'Marvel', '1964-01-01', '/covers/b.jpg', '', 201, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}
	// Row C: never synced -> always selected.
	if _, err := db.Exec(`INSERT INTO comics (series, comic_vine_id, comic_vine_synced_at) VALUES ('C', 203, '')`); err != nil {
		t.Fatal(err)
	}
	// Row D: synced 40 days ago, past the 30-day default cooldown -> selected again.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, comic_vine_id, comic_vine_synced_at)
		VALUES ('D', 'DC', '1980-01-01', '/covers/d.jpg', '', 204, ?)`, longAgoSynced); err != nil {
		t.Fatal(err)
	}
	// Row E: everything filled in -> not selected regardless of cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, comic_vine_id, comic_vine_synced_at)
		VALUES ('E', 'Image', '2020-01-01', '/covers/e.jpg', 'Complete', 205, '')`); err != nil {
		t.Fatal(err)
	}

	settings := defaultComicVineComicScanSettings()
	rows, err := selectIncompleteComicVineComics(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}

	got := map[int]bool{}
	for _, r := range rows {
		got[r.ID] = true
	}
	if got[1] {
		t.Fatal("comic without a Comic Vine ID should never be selected")
	}
	if got[2] {
		t.Fatal("recently-synced comic should be skipped during cooldown")
	}
	if !got[3] {
		t.Fatal("never-synced comic should always be selected")
	}
	if !got[4] {
		t.Fatal("comic synced past the cooldown window should be selected again")
	}
	if got[5] {
		t.Fatal("comic with no missing fields should not be selected")
	}
}

func TestEnrichIncompleteComicFromComicVinePreservesExistingValues(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	res, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, comic_vine_id)
		VALUES ('A', 'Existing Publisher', '', '', '', 301)`)
	if err != nil {
		t.Fatal(err)
	}
	id64, _ := res.LastInsertId()
	comicID := int(id64)

	issue := comicvine.Issue{
		ID:          301,
		CoverDate:   "2001-01-01",
		Description: "<p>An origin story.</p>",
	}
	covers := &CoverCache{}
	if err := enrichIncompleteComicFromComicVine(context.Background(), db, covers, comicID, issue, "New Publisher"); err != nil {
		t.Fatal(err)
	}

	var publisher, coverDate, description, syncedAt string
	if err := db.Get(&publisher, `SELECT publisher FROM comics WHERE id = ?`, comicID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&coverDate, `SELECT cover_date FROM comics WHERE id = ?`, comicID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&description, `SELECT description FROM comics WHERE id = ?`, comicID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&syncedAt, `SELECT comic_vine_synced_at FROM comics WHERE id = ?`, comicID); err != nil {
		t.Fatal(err)
	}
	if publisher != "Existing Publisher" {
		t.Fatalf("publisher = %q; want existing value preserved", publisher)
	}
	if coverDate != "2001-01-01" {
		t.Fatalf("cover_date = %q; want value filled from Comic Vine", coverDate)
	}
	if description != "<p>An origin story.</p>" {
		t.Fatalf("description = %q; want value filled from Comic Vine", description)
	}
	if syncedAt == "" {
		t.Fatal("comic_vine_synced_at should be stamped after a successful enrich")
	}
}

func TestComicVineComicScanTriggerRequiresAPIKey(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	settings := defaultComicVineComicScanSettings()
	settings.Enabled = true
	if err := saveComicVineComicScanSettings(context.Background(), db, settings); err != nil {
		t.Fatal(err)
	}

	client := comicvine.NewClient("")
	scanner := NewComicVineComicScanner(db, client, &CoverCache{})

	if err := scanner.trigger("manual"); err == nil {
		t.Fatal("expected trigger to fail without a configured Comic Vine API key")
	}

	status := scanner.snapshot(context.Background())
	if status.APIKeyConfigured {
		t.Fatal("expected APIKeyConfigured to be false for an empty API key")
	}
}

func TestComicVineComicScanSnapshotReportsAPIKeyConfigured(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	client := comicvine.NewClient("a-real-key")
	scanner := NewComicVineComicScanner(db, client, &CoverCache{})

	status := scanner.snapshot(context.Background())
	if !status.APIKeyConfigured {
		t.Fatal("expected APIKeyConfigured to be true when a key is set")
	}
}

func TestComicVineComicScanEnrichesFromMockedAPI(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	if _, err := db.Exec(`INSERT INTO comics (series, comic_vine_id) VALUES ('A', 501)`); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/issue/4000-501/":
			_ = json.NewEncoder(w).Encode(comicvine.SingleIssueResponse{
				StatusCode: 1,
				Results: comicvine.Issue{
					ID:          501,
					CoverDate:   "1999-06-01",
					Description: "First appearance.",
					Image:       comicvine.Image{OriginalURL: ""},
					Volume:      &comicvine.Volume{ID: 900, Name: "Test Volume"},
				},
			})
		case r.URL.Path == "/volume/4050-900/":
			_ = json.NewEncoder(w).Encode(comicvine.SingleVolumeResponse{
				StatusCode: 1,
				Results: comicvine.Volume{
					ID:        900,
					Name:      "Test Volume",
					Publisher: comicvine.Publisher{Name: "Test Publisher"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := comicvine.NewClient("test-key", comicvine.WithBaseURL(server.URL))
	covers := &CoverCache{}
	scanner := NewComicVineComicScanner(db, client, covers)

	settings := defaultComicVineComicScanSettings()
	rows, err := selectIncompleteComicVineComics(context.Background(), db, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 eligible comic, got %d", len(rows))
	}

	budget := &comicVineMaintenanceBudget{db: db, limit: settings.DailyCallLimit}
	if err := scanner.scanIncompleteComicVineComics(context.Background(), rows, settings, budget); err != nil {
		t.Fatal(err)
	}

	var publisher, coverDate, description string
	if err := db.Get(&publisher, `SELECT publisher FROM comics WHERE id = ?`, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&coverDate, `SELECT cover_date FROM comics WHERE id = ?`, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&description, `SELECT description FROM comics WHERE id = ?`, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	if publisher != "Test Publisher" {
		t.Fatalf("publisher = %q; want %q", publisher, "Test Publisher")
	}
	if coverDate != "1999-06-01" {
		t.Fatalf("cover_date = %q; want %q", coverDate, "1999-06-01")
	}
	if description != "First appearance." {
		t.Fatalf("description = %q; want %q", description, "First appearance.")
	}
}

func TestComicVineComicScanDailyQuotaIsShared(t *testing.T) {
	db := newComicVineComicScannerTestDB(t)
	now := time.Now()
	for i := 0; i < 3; i++ {
		claimed, err := claimComicVineComicScanCall(context.Background(), db, 2, now)
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 && !claimed {
			t.Fatalf("call %d should have been claimed", i)
		}
		if i == 2 && claimed {
			t.Fatal("third call should have exceeded the daily limit")
		}
	}
	usage := currentComicVineComicScanUsage(context.Background(), db, now)
	if usage.Calls != 2 {
		t.Fatalf("usage.Calls = %d; want 2", usage.Calls)
	}
}
