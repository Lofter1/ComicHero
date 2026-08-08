package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Lofter1/ComicHero/backend/comicvine"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type ComicVineRefreshComicInput struct {
	ID int `path:"id" doc:"Local comic identifier." example:"42"`
}

// RegisterComicVineComicRoutes wires up the on-demand, single-comic Comic Vine
// refresh endpoint. This is separate from the Comic Vine maintenance scan:
// it always runs immediately for one comic and overwrites existing fields,
// rather than filling blanks in the background on a quota-limited schedule.
func RegisterComicVineComicRoutes(api huma.API, db *sqlx.DB, client *comicvine.Client, covers *CoverCache) {
	huma.Register(api, huma.Operation{
		OperationID: "refreshComicFromComicVine",
		Tags:        []string{tagComicVine},
		Summary:     "Refresh comic from Comic Vine",
		Description: "Re-fetches metadata for a comic from its already-linked Comic Vine issue and overwrites the comic's fields, while preserving local read status.",
		Method:      http.MethodPatch,
		Path:        "/comics/{id}/comicvine",
		Errors:      []int{400, 401, 404, 500, 502},
	}, func(ctx context.Context, input *ComicVineRefreshComicInput) (*ComicDetailOutput, error) {
		if _, err := currentUserID(ctx); err != nil {
			return nil, err
		}
		if client == nil || !client.HasAPIKey() {
			return nil, huma.Error400BadRequest("no Comic Vine API key is configured")
		}

		var comicVineID sql.NullInt64
		if err := db.GetContext(ctx, &comicVineID, `SELECT comic_vine_id FROM comics WHERE id = ?`, input.ID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("comic not found")
			}
			return nil, huma.Error500InternalServerError("failed to look up comic")
		}
		if !comicVineID.Valid || comicVineID.Int64 <= 0 {
			log.Printf("Comic Vine comic refresh rejected: comic_id=%d comic_vine_id_valid=%v comic_vine_id=%d", input.ID, comicVineID.Valid, comicVineID.Int64)
			return nil, huma.Error400BadRequest(fmt.Sprintf("comic has no linked Comic Vine issue (comic_vine_id=%d)", comicVineID.Int64))
		}

		issue, err := client.Issues().GetByID(ctx, int(comicVineID.Int64), []string{})
		if err != nil {
			return nil, comicVineAPIError(err)
		}

		var volume *comicvine.Volume
		if issue.Volume != nil && issue.Volume.ID > 0 {
			fetched, volumeErr := client.Volumes().GetByID(ctx, issue.Volume.ID, []string{"name", "publisher", "start_year"})
			if volumeErr != nil {
				// Best-effort: the issue's own fields are still worth applying
				// even if the volume (and thus publisher/series) lookup fails.
				log.Printf("Comic Vine comic refresh: comic_id=%d comic_vine_id=%d stage=volume: %v", input.ID, comicVineID.Int64, volumeErr)
			} else {
				volume = fetched
			}
		}

		return updateComicFromComicVine(ctx, db, covers, input.ID, *issue, volume)
	})
}

// updateComicFromComicVine overwrites a comic's series, issue, publisher,
// cover date, cover image, and description with fresh data from Comic Vine.
// Fields Comic Vine didn't return (e.g. the volume lookup failed, or Comic
// Vine itself has that field blank) are left as they were rather than wiped.
func updateComicFromComicVine(ctx context.Context, db *sqlx.DB, covers *CoverCache, comicID int, issue comicvine.Issue, volume *comicvine.Volume) (*ComicDetailOutput, error) {
	println(issue.ID)
	coverImage, err := localCoverURL(ctx, covers, comicVineIssueCoverSource(issue))
	if err != nil {
		log.Printf("Comic Vine comic refresh failed: comic_id=%d comic_vine_id=%d stage=cover: %v", comicID, issue.ID, err)
		return nil, err
	}

	series, publisher := "", ""
	seriesYear := 0
	if volume != nil {
		series = strings.TrimSpace(volume.Name)
		publisher = strings.TrimSpace(volume.Publisher.Name)
		if year, err := strconv.Atoi(strings.TrimSpace(volume.StartYear)); err == nil {
			seriesYear = year
		}
	}

	result, err := db.ExecContext(ctx, `
		UPDATE comics SET
			series = CASE WHEN ? <> '' THEN ? ELSE series END,
			series_year = CASE WHEN ? > 0 THEN ? ELSE series_year END,
			issue = CASE WHEN ? <> '' THEN ? ELSE issue END,
			publisher = CASE WHEN ? <> '' THEN ? ELSE publisher END,
			cover_date = CASE WHEN ? <> '' THEN ? ELSE cover_date END,
			cover_image = CASE WHEN ? <> '' THEN ? ELSE cover_image END,
			description = CASE WHEN ? <> '' THEN ? ELSE description END,
			comic_vine_id = ?,
			comic_vine_synced_at = ?
		WHERE id = ?
	`,
		series, series,
		seriesYear, seriesYear,
		strings.TrimSpace(issue.IssueNumber), strings.TrimSpace(issue.IssueNumber),
		publisher, publisher,
		issue.CoverDate, issue.CoverDate,
		coverImage, coverImage,
		issue.Description, issue.Description,
		issue.ID,
		time.Now().UTC().Format(time.RFC3339),
		comicID,
	)
	if err != nil {
		log.Printf("Comic Vine comic refresh failed: comic_id=%d comic_vine_id=%d stage=metadata: %v", comicID, issue.ID, err)
		return nil, huma.Error500InternalServerError("failed to update comic from Comic Vine")
	}
	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}

	output, err := getComic(ctx, db, comicID)
	if err != nil {
		log.Printf("Comic Vine comic refresh failed: comic_id=%d comic_vine_id=%d stage=response: %v", comicID, issue.ID, err)
		return nil, err
	}
	return output, nil
}

// comicVineAPIError maps a Comic Vine API error to an HTTP response. Comic Vine
// reuses the StatusCode field for two different things depending on how the
// error occurred: a real HTTP status (>= 400) when the transport-level request
// failed, or one of Comic Vine's own documented status codes (e.g. 100 = invalid
// API key, 101 = object not found) when the request succeeded at the HTTP level
// but Comic Vine reported a logical error in the response body. Both are handled
// here since either can come back from the same call.
func comicVineAPIError(err error) error {
	apiErr, ok := err.(*comicvine.APIError)
	if !ok {
		return huma.Error502BadGateway(fmt.Sprintf("Comic Vine request failed: %v", err))
	}
	switch apiErr.StatusCode {
	case http.StatusNotFound, 101: // HTTP 404, or Comic Vine's "Object Not Found"
		return huma.Error404NotFound("Comic Vine issue not found")
	case http.StatusUnauthorized, http.StatusForbidden, 100: // HTTP 401/403, or Comic Vine's "Invalid API Key"
		return huma.Error502BadGateway("Comic Vine rejected the configured API key")
	case http.StatusTooManyRequests, 107: // HTTP 429, or Comic Vine's rate-limit code
		return huma.Error502BadGateway("Comic Vine rate limit exceeded, try again later")
	default:
		return huma.Error502BadGateway(fmt.Sprintf("Comic Vine request failed: %s", apiErr.Message))
	}
}
