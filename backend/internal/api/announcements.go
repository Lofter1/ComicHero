package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const announcementSettingsKey = "user_announcement"

func getAnnouncementSettings(ctx context.Context, db *sqlx.DB) (*AnnouncementSettings, error) {
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, announcementSettingsKey); err != nil {
		if err == sql.ErrNoRows {
			return &AnnouncementSettings{Level: "info"}, nil
		}
		return nil, err
	}
	settings := &AnnouncementSettings{}
	if err := json.Unmarshal([]byte(value), settings); err != nil {
		return nil, err
	}
	if settings.Level == "" {
		settings.Level = "info"
	}
	return settings, nil
}

func announcementIsActive(settings *AnnouncementSettings, now time.Time) bool {
	if settings == nil || !settings.Enabled || strings.TrimSpace(settings.Message) == "" {
		return false
	}
	if settings.StartsAt != "" {
		starts, err := time.Parse(time.RFC3339, settings.StartsAt)
		if err != nil || now.Before(starts) {
			return false
		}
	}
	if settings.EndsAt != "" {
		ends, err := time.Parse(time.RFC3339, settings.EndsAt)
		if err != nil || !now.Before(ends) {
			return false
		}
	}
	return true
}

func getActiveAnnouncement(ctx context.Context, db *sqlx.DB) (*AnnouncementOutput, error) {
	settings, err := getAnnouncementSettings(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch announcement")
	}
	if !announcementIsActive(settings, time.Now().UTC()) {
		return &AnnouncementOutput{Body: nil}, nil
	}
	return &AnnouncementOutput{Body: settings}, nil
}

func getAdminAnnouncement(ctx context.Context, db *sqlx.DB) (*AnnouncementOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	settings, err := getAnnouncementSettings(ctx, db)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch announcement settings")
	}
	return &AnnouncementOutput{Body: settings}, nil
}

func updateAnnouncement(ctx context.Context, db *sqlx.DB, settings AnnouncementSettings) (*AnnouncementOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	settings.Message = strings.TrimSpace(settings.Message)
	if len(settings.Message) > 4000 {
		return nil, huma.Error400BadRequest("announcement is too long")
	}
	if settings.Level != "info" && settings.Level != "warning" && settings.Level != "critical" {
		return nil, huma.Error400BadRequest("announcement level must be info, warning, or critical")
	}
	if settings.StartsAt != "" {
		if _, err := time.Parse(time.RFC3339, settings.StartsAt); err != nil {
			return nil, huma.Error400BadRequest("announcement start time must be RFC3339")
		}
	}
	if settings.EndsAt != "" {
		if _, err := time.Parse(time.RFC3339, settings.EndsAt); err != nil {
			return nil, huma.Error400BadRequest("announcement end time must be RFC3339")
		}
	}
	if settings.StartsAt != "" && settings.EndsAt != "" {
		starts, _ := time.Parse(time.RFC3339, settings.StartsAt)
		ends, _ := time.Parse(time.RFC3339, settings.EndsAt)
		if !starts.Before(ends) {
			return nil, huma.Error400BadRequest("announcement end time must be after its start time")
		}
	}
	settings.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(settings)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to encode announcement")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, announcementSettingsKey, string(encoded)); err != nil {
		return nil, huma.Error500InternalServerError("failed to save announcement")
	}
	return &AnnouncementOutput{Body: &settings}, nil
}
