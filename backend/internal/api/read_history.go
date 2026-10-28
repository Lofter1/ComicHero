package api

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const readHistoryFormatVersion = 1

func exportReadHistory(ctx context.Context, db *sqlx.DB) (*ReadHistoryExportOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}

	comics := []ReadHistoryComic{}
	if err := db.SelectContext(ctx, &comics, `
		SELECT c.id, c.metron_issue_id, c.comic_vine_id, c.series, c.series_year,
		       c.issue, c.publisher, c.cover_date, uc.read_at
		FROM comics c
		JOIN user_comics uc ON uc.comic_id = c.id
		WHERE uc.user_id = ? AND uc.read = 1
		ORDER BY c.series, c.series_year, c.issue, c.id
	`, userID); err != nil {
		return nil, huma.Error500InternalServerError("failed to export read history")
	}

	return &ReadHistoryExportOutput{Body: ReadHistoryExport{
		FormatVersion: readHistoryFormatVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Comics:        comics,
	}}, nil
}

func importReadHistory(ctx context.Context, db *sqlx.DB, input ReadHistoryExport) (*ReadHistoryImportOutput, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, err
	}
	if input.FormatVersion != readHistoryFormatVersion {
		return nil, huma.Error400BadRequest("unsupported read-history format version")
	}
	if len(input.Comics) > 100000 {
		return nil, huma.Error400BadRequest("read-history export is too large")
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import read history")
	}
	defer func() { _ = tx.Rollback() }()

	result := ReadHistoryImportResult{Unmatched: []ReadHistoryUnmatched{}}
	for _, item := range input.Comics {
		comicID, ok, err := findReadHistoryComic(ctx, tx, item)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to match read history")
		}
		if !ok {
			result.Unmatched = append(result.Unmatched, ReadHistoryUnmatched{Comic: item, Reason: "no matching comic found"})
			continue
		}

		readAt := strings.TrimSpace(item.ReadAt)
		if readAt == "" {
			readAt = time.Now().UTC().Format(time.RFC3339)
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_comics (comic_id, user_id, read, read_at)
			VALUES (?, ?, 1, ?)
			ON CONFLICT(comic_id, user_id) DO UPDATE SET
				read = 1,
				read_at = CASE WHEN user_comics.read_at = '' OR excluded.read_at > user_comics.read_at
				              THEN excluded.read_at ELSE user_comics.read_at END
		`, comicID, userID, readAt)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to import read history")
		}
		result.Imported++
	}
	if err := tx.Commit(); err != nil {
		return nil, huma.Error500InternalServerError("failed to import read history")
	}
	return &ReadHistoryImportOutput{Body: result}, nil
}

func findReadHistoryComic(ctx context.Context, tx *sqlx.Tx, item ReadHistoryComic) (int, bool, error) {
	var id int
	var err error
	// Stable upstream IDs take precedence over all local fields. Do not combine
	// the two IDs in one OR query: a stale/mistyped pair could identify two comics.
	if item.MetronIssueID != nil && *item.MetronIssueID > 0 {
		err = tx.GetContext(ctx, &id, `SELECT id FROM comics WHERE metron_issue_id = ?`, *item.MetronIssueID)
	} else if item.ComicVineID != nil && *item.ComicVineID > 0 {
		err = tx.GetContext(ctx, &id, `SELECT id FROM comics WHERE comic_vine_id = ?`, *item.ComicVineID)
	} else {
		var ids []int
		err = tx.SelectContext(ctx, &ids, `
			SELECT id FROM comics
			WHERE series = ? AND series_year = ? AND issue = ? AND publisher = ?
			ORDER BY id LIMIT 2
		`, item.Series, item.SeriesYear, item.Issue, item.Publisher)
		if err == nil {
			if len(ids) != 1 {
				return 0, false, nil
			}
			id = ids[0]
		}
	}
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	return id, err == nil, err
}
