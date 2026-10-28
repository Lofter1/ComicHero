package api

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newAnnouncementTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE users (id INTEGER PRIMARY KEY, is_admin INTEGER NOT NULL DEFAULT 0);
		INSERT INTO users (id, is_admin) VALUES (1, 1), (2, 0);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestAnnouncementOnlyReturnsWithinSchedule(t *testing.T) {
	db := newAnnouncementTestDB(t)
	ctx := context.Background()
	starts := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	ends := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if _, err := updateAnnouncement(context.WithValue(ctx, contextUserIDKey{}, 1), db, AnnouncementSettings{
		Enabled: true, Message: "Maintenance soon", Level: "warning", StartsAt: starts, EndsAt: ends,
	}); err != nil {
		t.Fatalf("save announcement: %v", err)
	}
	active, err := getActiveAnnouncement(ctx, db)
	if err != nil || active.Body == nil || active.Body.Message != "Maintenance soon" {
		t.Fatalf("active announcement = %#v, err %v", active.Body, err)
	}

	if _, err := updateAnnouncement(context.WithValue(ctx, contextUserIDKey{}, 1), db, AnnouncementSettings{
		Enabled: true, Message: "Later", Level: "info", StartsAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save future announcement: %v", err)
	}
	inactive, err := getActiveAnnouncement(ctx, db)
	if err != nil || inactive.Body != nil {
		t.Fatalf("future announcement = %#v, err %v; want nil", inactive.Body, err)
	}
}

func TestAnnouncementUpdateRequiresAdmin(t *testing.T) {
	db := newAnnouncementTestDB(t)
	_, err := updateAnnouncement(context.WithValue(context.Background(), contextUserIDKey{}, 2), db, AnnouncementSettings{
		Enabled: true, Message: "Nope", Level: "info",
	})
	if err == nil {
		t.Fatal("non-admin update returned nil error")
	}
}
