package api

import (
	"context"
	"database/sql"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type comicMergeRow struct {
	ID             int    `db:"id"`
	SeriesID       *int   `db:"series_id"`
	Series         string `db:"series"`
	SeriesYear     int    `db:"series_year"`
	Issue          string `db:"issue"`
	Publisher      string `db:"publisher"`
	CoverDate      string `db:"cover_date"`
	CoverImage     string `db:"cover_image"`
	Description    string `db:"description"`
	MetronIssueID  *int   `db:"metron_issue_id"`
	ComicVineID    *int   `db:"comic_vine_id"`
	MetronSyncedAt string `db:"metron_synced_at"`
}

func mergeComic(ctx context.Context, db *sqlx.DB, targetID, sourceID int) (*ComicDetailOutput, error) {
	if _, err := requireAdminUser(ctx, db); err != nil {
		return nil, err
	}
	if targetID <= 0 || sourceID <= 0 {
		return nil, huma.Error400BadRequest("comic IDs must be positive")
	}
	if targetID == sourceID {
		return nil, huma.Error400BadRequest("a comic cannot be merged into itself")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to start comic merge")
	}
	defer tx.Rollback()

	target, err := getComicMergeRow(ctx, tx, targetID)
	if err != nil {
		return nil, err
	}
	source, err := getComicMergeRow(ctx, tx, sourceID)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_comics (comic_id, user_id, read, skipped, read_at)
		SELECT ?, user_id, read, skipped, read_at
		FROM user_comics
		WHERE comic_id = ?
		ON CONFLICT(comic_id, user_id) DO UPDATE SET
			read = MAX(user_comics.read, excluded.read),
			skipped = MAX(user_comics.skipped, excluded.skipped),
			read_at = CASE
				WHEN user_comics.read_at = '' THEN excluded.read_at
				WHEN excluded.read_at = '' THEN user_comics.read_at
				WHEN excluded.read_at > user_comics.read_at THEN excluded.read_at
				ELSE user_comics.read_at
			END
	`, target.ID, source.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to merge comic user state")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_comics WHERE comic_id = ?`, source.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to remove duplicate comic user state")
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO comic_characters (comic_id, character_id)
		SELECT ?, character_id FROM comic_characters WHERE comic_id = ?
	`, target.ID, source.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to merge comic characters")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM comic_characters WHERE comic_id = ?`, source.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to remove duplicate comic characters")
	}

	for _, table := range []string{"reading_order_comics", "arc_comics"} {
		if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET comic_id = ? WHERE comic_id = ?`, target.ID, source.ID); err != nil {
			return nil, huma.Error500InternalServerError("failed to merge comic placements")
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM comics WHERE id = ?`, source.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to delete duplicate comic")
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE comics SET
			series_id = COALESCE(series_id, ?),
			series = CASE WHEN TRIM(series) = '' THEN ? ELSE series END,
			series_year = CASE WHEN series_year = 0 THEN ? ELSE series_year END,
			issue = CASE WHEN TRIM(issue) = '' THEN ? ELSE issue END,
			publisher = CASE WHEN TRIM(publisher) = '' THEN ? ELSE publisher END,
			cover_date = CASE WHEN TRIM(cover_date) = '' THEN ? ELSE cover_date END,
			cover_image = CASE WHEN TRIM(cover_image) = '' THEN ? ELSE cover_image END,
			description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END,
			metron_issue_id = COALESCE(metron_issue_id, ?),
			comic_vine_id = COALESCE(comic_vine_id, ?),
			metron_synced_at = CASE
				WHEN metron_synced_at = '' THEN ?
				WHEN ? > metron_synced_at THEN ?
				ELSE metron_synced_at
			END
		WHERE id = ?
	`, source.SeriesID, strings.TrimSpace(source.Series), source.SeriesYear, strings.TrimSpace(source.Issue),
		strings.TrimSpace(source.Publisher), source.CoverDate, source.CoverImage, source.Description,
		source.MetronIssueID, source.ComicVineID, source.MetronSyncedAt, source.MetronSyncedAt,
		source.MetronSyncedAt, target.ID); err != nil {
		return nil, huma.Error500InternalServerError("failed to merge comic metadata")
	}

	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to commit comic merge")
	}
	return getComic(ctx, db, target.ID)
}

func getComicMergeRow(ctx context.Context, tx *sqlx.Tx, id int) (comicMergeRow, error) {
	var comic comicMergeRow
	err := tx.GetContext(ctx, &comic, `
		SELECT id, series_id, series, series_year, issue, publisher, cover_date, cover_image,
			description, metron_issue_id, comic_vine_id, metron_synced_at
		FROM comics
		WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return comicMergeRow{}, huma.Error404NotFound("comic not found")
	}
	if err != nil {
		return comicMergeRow{}, huma.Error500InternalServerError("failed to fetch comic for merge")
	}
	return comic, nil
}
