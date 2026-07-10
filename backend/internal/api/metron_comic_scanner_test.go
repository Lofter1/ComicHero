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
