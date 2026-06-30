package api

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const (
	metronResourceArc         = "arc"
	metronResourceCharacter   = "character"
	metronResourceIssue       = "issue"
	metronResourceReadingList = "reading_list"
	metronResourceSeries      = "series"
)

type metronSyncState struct {
	LastModified string `db:"last_modified"`
	FullySynced  bool   `db:"fully_synced"`
}

func metronConditional(ctx context.Context, db *sqlx.DB, resourceType string, metronID int, force bool) (metron.ConditionalRequest, error) {
	if force || metronID <= 0 {
		return metron.ConditionalRequest{Force: force}, nil
	}

	var state metronSyncState
	if err := db.GetContext(ctx, &state, `
		SELECT last_modified, fully_synced
		FROM metron_sync_states
		WHERE resource_type = ? AND metron_id = ?
	`, resourceType, metronID); err != nil {
		if isNoRows(err) {
			return metron.ConditionalRequest{}, nil
		}
		return metron.ConditionalRequest{}, huma.Error500InternalServerError("failed to read Metron sync state")
	}

	if !state.FullySynced || state.LastModified == "" {
		return metron.ConditionalRequest{}, nil
	}
	return metron.ConditionalRequest{LastModified: state.LastModified}, nil
}

func markMetronSynced(ctx context.Context, db *sqlx.DB, resourceType string, metronID int, info metron.FetchInfo) error {
	if metronID <= 0 {
		return nil
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO metron_sync_states (resource_type, metron_id, last_modified, fully_synced, synced_at)
		VALUES (?, ?, ?, 1, ?)
		ON CONFLICT(resource_type, metron_id) DO UPDATE SET
			last_modified = CASE WHEN excluded.last_modified <> '' THEN excluded.last_modified ELSE metron_sync_states.last_modified END,
			fully_synced = 1,
			synced_at = excluded.synced_at
	`, resourceType, metronID, info.LastModified, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return huma.Error500InternalServerError("failed to save Metron sync state")
	}
	return nil
}

func markMetronNotModified(ctx context.Context, db *sqlx.DB, resourceType string, metronID int) error {
	if metronID <= 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, `
		UPDATE metron_sync_states
		SET synced_at = ?
		WHERE resource_type = ? AND metron_id = ?
	`, time.Now().UTC().Format(time.RFC3339), resourceType, metronID)
	if err != nil {
		return huma.Error500InternalServerError("failed to update Metron sync state")
	}
	return nil
}

func fetchMetronIssue(ctx context.Context, db *sqlx.DB, client *metron.Client, metronID int, force bool) (*metron.Issue, metron.FetchInfo, error) {
	conditional, err := metronConditional(ctx, db, metronResourceIssue, metronID, force)
	if err != nil {
		return nil, metron.FetchInfo{}, err
	}
	issue, info, err := client.GetIssueConditional(ctx, metronID, conditional)
	return issue, info, err
}

func fetchMetronSeries(ctx context.Context, db *sqlx.DB, client *metron.Client, metronID int, force bool) (*metron.Series, metron.FetchInfo, error) {
	conditional, err := metronConditional(ctx, db, metronResourceSeries, metronID, force)
	if err != nil {
		return nil, metron.FetchInfo{}, err
	}
	series, info, err := client.GetSeriesConditional(ctx, metronID, conditional)
	return series, info, err
}

func fetchMetronReadingList(ctx context.Context, db *sqlx.DB, client *metron.Client, metronID int, force bool) (*metron.ReadingList, metron.FetchInfo, error) {
	conditional, err := metronConditional(ctx, db, metronResourceReadingList, metronID, force)
	if err != nil {
		return nil, metron.FetchInfo{}, err
	}
	list, info, err := client.GetReadingListConditional(ctx, metronID, conditional)
	return list, info, err
}

func fetchMetronArc(ctx context.Context, db *sqlx.DB, client *metron.Client, metronID int, force bool) (*metron.MetronArc, metron.FetchInfo, error) {
	conditional, err := metronConditional(ctx, db, metronResourceArc, metronID, force)
	if err != nil {
		return nil, metron.FetchInfo{}, err
	}
	arc, info, err := client.GetArcConditional(ctx, metronID, conditional)
	return arc, info, err
}

func fetchMetronCharacter(ctx context.Context, db *sqlx.DB, client *metron.Client, metronID int, force bool) (*metron.MetronCharacter, metron.FetchInfo, error) {
	conditional, err := metronConditional(ctx, db, metronResourceCharacter, metronID, force)
	if err != nil {
		return nil, metron.FetchInfo{}, err
	}
	character, info, err := client.GetCharacterConditional(ctx, metronID, conditional)
	return character, info, err
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
