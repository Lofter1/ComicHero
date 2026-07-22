package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

const metronIssueAlreadyLinkedProblem = "urn:comichero:problem:metron-issue-already-linked"

func registerMetronComicRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
	huma.Register(api, huma.Operation{
		OperationID: "searchMetronComics",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron comics",
		Description: "Searches Metron for comic issue metadata. An exact Comic Vine ID takes precedence; when series is omitted, q is sent as the Metron series-name search.",
		Method:      http.MethodGet,
		Path:        "/metron/comics",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIssueListInput) (*MetronIssueListOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/comics"); err != nil {
			return nil, err
		}
		issues, err := searchMetronIssues(ctx, client, input)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronIssueListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: issues}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronComic",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron comic",
		Description: "Gets a Metron comic issue by ID.",
		Method:      http.MethodGet,
		Path:        "/metron/comics/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronIssueOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeDetail, "GET /metron/comics/{id}"); err != nil {
			return nil, err
		}
		issue, err := client.GetIssue(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronIssueOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *issue}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronComic",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron comic",
		Description:   "Starts a background job that imports a Metron comic issue for use in reading orders. If the issue is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/comics/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/comics/{id}/import"); err != nil {
			return nil, err
		}
		job := startMetronComicImportWithOptions(ctx, importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateComicFromMetron",
		Tags:        []string{tagMetron},
		Summary:     "Update comic from Metron",
		Description: "Updates an existing comic's metadata from a Metron issue while preserving local read status.",
		Method:      http.MethodPatch,
		Path:        "/comics/{id}/metron",
		Errors:      errsMetronSync,
	}, func(ctx context.Context, input *UpdateComicFromMetronInput) (*ComicDetailOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "PATCH /comics/{id}/metron"); err != nil {
			return nil, err
		}
		if !input.Body.MergeDuplicate {
			if err := rejectMetronIssueLinkedToAnotherComic(ctx, db, input.ID, input.Body.MetronIssueID); err != nil {
				return nil, err
			}
		}
		issue, info, err := fetchMetronIssue(ctx, db, client, input.Body.MetronIssueID, input.Body.Force)
		if err != nil {
			return nil, metronAPIError(err)
		}
		var merged *ComicDetailOutput
		if input.Body.MergeDuplicate {
			merged, err = mergeComicLinkedToMetronIssue(ctx, db, input.ID, input.Body.MetronIssueID)
			if err != nil {
				return nil, err
			}
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, input.Body.MetronIssueID); err != nil {
				return nil, err
			}
			if merged != nil {
				return withMetronRateLimit(merged, client.CurrentRateLimit()), nil
			}
			output, err := getComic(ctx, db, input.ID)
			if err != nil {
				return nil, err
			}
			return withMetronRateLimit(output, client.CurrentRateLimit()), nil
		}
		output, err := updateComicFromMetron(ctx, db, client, covers, input.ID, *issue)
		if err != nil {
			return nil, err
		}
		if err := markMetronSynced(ctx, db, metronResourceIssue, input.Body.MetronIssueID, info); err != nil {
			return nil, err
		}
		return withMetronRateLimit(output, client.CurrentRateLimit()), nil
	})
}

func searchMetronIssues(ctx context.Context, client *metron.Client, input *MetronIssueListInput) ([]metron.Issue, error) {
	if input.ComicVineID > 0 {
		return client.SearchIssuesByComicVineID(ctx, input.ComicVineID)
	}
	return client.SearchIssues(ctx, input.Query, input.Series, input.Issue)
}

func comicPayloadFromMetronIssue(issue metron.Issue) ComicPayload {
	return ComicPayload{
		Series:      issue.Series,
		SeriesYear:  issue.SeriesYear,
		Issue:       issue.Issue,
		Publisher:   issue.Publisher,
		CoverDate:   issue.CoverDate,
		CoverImage:  issue.CoverImage,
		Description: issue.Description,
	}
}

func importMetronComic(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue) (*ComicDetailOutput, error) {
	return importMetronComicWithOptions(ctx, db, client, covers, issue, defaultMetronImportOptions())
}

func importMetronComicWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	if id, ok, err := existingComicIDForMetronIssue(ctx, db, issue); err != nil {
		return nil, err
	} else if ok {
		return reuseComicForMetronIssue(ctx, db, client, covers, id, issue, options)
	}

	return createMetronComicWithOptions(ctx, db, client, covers, issue, options)
}

func reuseComicForMetronIssue(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, comicID int, issue metron.Issue, options MetronImportOptions) (*ComicDetailOutput, error) {
	if err := linkComicToMetronIssueIdentity(ctx, db, comicID, issue); err != nil {
		return nil, err
	}
	if options.Force {
		return updateComicFromMetron(ctx, db, client, covers, comicID, issue)
	}
	if err := syncMetronIssueArcsWithOptions(ctx, db, client, comicID, issue, options); err != nil {
		return nil, err
	}
	if options.includesCharacters() {
		if err := syncMetronIssueCharactersWithOptions(ctx, db, client, covers, comicID, issue, options); err != nil {
			return nil, err
		}
	}
	return getComic(ctx, db, comicID)
}

func linkComicToMetronIssueIdentity(ctx context.Context, db *sqlx.DB, comicID int, issue metron.Issue) error {
	if issue.ID > 0 {
		if err := attachMetronIssueID(ctx, db, comicID, issue.ID); err != nil {
			return err
		}
	}
	if issue.ComicVineID > 0 {
		if err := attachComicVineID(ctx, db, comicID, issue.ComicVineID); err != nil {
			return err
		}
	}
	return linkComicToMetronIssueSeries(ctx, db, comicID, issue)
}

func importMetronComicSweep(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions, fetchIssueDetail bool) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	var issueInfo metron.FetchInfo
	if options.needsIssueDetail() && fetchIssueDetail && client != nil && issue.ID > 0 {
		if !options.Force {
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				complete, err := comicHasRequestedMetronData(ctx, db, id, options)
				if err != nil {
					return nil, err
				}
				if complete {
					return getComic(ctx, db, id)
				}
			}
		}
		forceIssue := options.Force
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
			return nil, err
		} else if ok {
			complete, err := comicHasRequestedMetronData(ctx, db, id, options)
			if err != nil {
				return nil, err
			}
			forceIssue = forceIssue || !complete
		}
		detail, info, err := fetchMetronIssue(ctx, db, client, issue.ID, forceIssue)
		if err != nil {
			if isContextCanceledError(err) {
				return nil, err
			}
			return nil, huma.Error502BadGateway(err.Error())
		}
		issueInfo = info
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, issue.ID); err != nil {
				return nil, err
			}
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				return getComic(ctx, db, id)
			}
			detail, issueInfo, err = fetchMetronIssue(ctx, db, client, issue.ID, true)
			if err != nil {
				return nil, huma.Error502BadGateway(err.Error())
			}
			issue = *detail
		} else {
			issue = *detail
		}
	}

	comic, err := importMetronComicWithOptions(ctx, db, client, covers, issue, options)
	if err != nil {
		return nil, err
	}
	if options.includesComics() && fetchIssueDetail && issue.ID > 0 {
		if err := markMetronSynced(ctx, db, metronResourceIssue, issue.ID, issueInfo); err != nil {
			return nil, err
		}
	}
	if options.includesSeries() && client != nil && issue.SeriesID > 0 {
		metadata, info, err := fetchMetronSeries(ctx, db, client, issue.SeriesID, options.Force)
		if err != nil {
			if isContextCanceledError(err) {
				return nil, err
			}
			return nil, huma.Error502BadGateway(err.Error())
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, issue.SeriesID); err != nil {
				return nil, err
			}
		} else {
			if err := updateImportedSeriesMetadata(ctx, db, *metadata); err != nil {
				return nil, err
			}
			if err := markMetronSynced(ctx, db, metronResourceSeries, issue.SeriesID, info); err != nil {
				return nil, err
			}
		}
	}
	return comic, nil
}

func existingComicIDByMetronIssueID(ctx context.Context, db *sqlx.DB, metronID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM comics WHERE metron_issue_id = ?
	`, metronID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check imported comic")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func existingComicIDForMetronIssue(ctx context.Context, db *sqlx.DB, issue metron.Issue) (int, bool, error) {
	if issue.ID > 0 {
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil || ok {
			return id, ok, err
		}
	}
	if issue.ComicVineID > 0 {
		if id, ok, err := existingComicIDByComicVineID(ctx, db, issue.ComicVineID); err != nil || ok {
			return id, ok, err
		}
	}
	return existingComicIDByMetronIssueMatch(ctx, db, issue)
}

func existingComicIDByComicVineID(ctx context.Context, db *sqlx.DB, comicVineID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM comics WHERE comic_vine_id = ?
	`, comicVineID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check Comic Vine comic")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func existingComicIDByMetronIssueMatch(ctx context.Context, db *sqlx.DB, issue metron.Issue) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM comics
		WHERE metron_issue_id IS NULL
			AND series = ?
			AND series_year = ?
			AND issue = ?
			AND publisher = ?
		ORDER BY id
		LIMIT 1
	`, issue.Series, issue.SeriesYear, issue.Issue, issue.Publisher); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check matching comic")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func comicHasRequestedMetronData(ctx context.Context, db *sqlx.DB, comicID int, options MetronImportOptions) (bool, error) {
	if !options.needsIssueDetail() {
		return true, nil
	}
	var comic Comic
	if err := db.GetContext(ctx, &comic, `SELECT * FROM comics WHERE id = ?`, comicID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, huma.Error500InternalServerError("failed to check imported comic")
	}
	if options.includesComics() && (comic.Description == "" || comic.CoverImage == "" || comic.ComicVineID == nil) {
		return false, nil
	}
	if options.includesSeries() {
		var count int
		if err := db.GetContext(ctx, &count, `
			SELECT COUNT(*) FROM series
			WHERE name = ? AND series_year = ? AND (metron_series_id IS NOT NULL OR description <> '' OR issue_count > 0)
		`, comic.Series, comic.SeriesYear); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported series metadata")
		}
		if count == 0 {
			return false, nil
		}
	}
	if options.includesArcs() {
		var count int
		if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM arc_comics WHERE comic_id = ?`, comicID); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported arcs")
		}
		if count == 0 {
			return false, nil
		}
	}
	if options.includesCharacters() {
		var count int
		if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM comic_characters WHERE comic_id = ?`, comicID); err != nil {
			return false, huma.Error500InternalServerError("failed to check imported characters")
		}
		if count == 0 {
			return false, nil
		}
	}
	return true, nil
}

func attachMetronIssueID(ctx context.Context, db *sqlx.DB, comicID, metronID int) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE comics SET metron_issue_id = ? WHERE id = ? AND metron_issue_id IS NULL
	`, metronID, comicID); err != nil {
		return huma.Error500InternalServerError("failed to link comic to Metron")
	}
	return nil
}

func attachComicVineID(ctx context.Context, db *sqlx.DB, comicID, comicVineID int) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE comics SET comic_vine_id = ? WHERE id = ? AND comic_vine_id IS NULL
	`, comicVineID, comicID); err != nil {
		return huma.Error500InternalServerError("failed to link comic to Comic Vine")
	}
	return nil
}

func linkComicToMetronIssueSeries(ctx context.Context, db *sqlx.DB, comicID int, issue metron.Issue) error {
	seriesID, err := ensureMetronIssueSeriesRow(ctx, db, issue)
	if err != nil {
		return err
	}
	if seriesID == 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE comics SET series_id = ? WHERE id = ?
	`, seriesID, comicID); err != nil {
		return huma.Error500InternalServerError("failed to link comic to series")
	}
	return nil
}

func createMetronComicWithOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issue metron.Issue, options MetronImportOptions) (*ComicDetailOutput, error) {
	options = resolveMetronImportOptions(options)
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	seriesID, err := ensureMetronIssueSeriesRow(ctx, db, issue)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series_id, series, series_year, issue, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, nullableSeriesID(seriesID),
		payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		nullableMetronID(issue.ID),
		nullablePositiveID(issue.ComicVineID),
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron comic")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported comic id")
	}
	if payload.Read {
		if err := setComicReadStatusForCurrentUser(ctx, db, int(id), payload.Read, true, nil); err != nil {
			return nil, err
		}
	}
	if err := syncMetronIssueArcsWithOptions(ctx, db, client, int(id), issue, options); err != nil {
		return nil, err
	}
	if options.includesCharacters() {
		if err := syncMetronIssueCharactersWithOptions(ctx, db, client, covers, int(id), issue, options); err != nil {
			return nil, err
		}
	}
	return getComic(ctx, db, int(id))
}

func updateComicFromMetron(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, comicID int, issue metron.Issue) (*ComicDetailOutput, error) {
	if err := rejectMetronIssueLinkedToAnotherComic(ctx, db, comicID, issue.ID); err != nil {
		return nil, err
	}
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=cover: %v", comicID, issue.ID, err)
		return nil, err
	}

	seriesID, err := ensureMetronIssueSeriesRow(ctx, db, issue)
	if err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=series: %v", comicID, issue.ID, err)
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series_id = ?, series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?, metron_issue_id = COALESCE(?, metron_issue_id), comic_vine_id = COALESCE(?, comic_vine_id)
		WHERE id = ?
	`, nullableSeriesID(seriesID),
		payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		nullableMetronID(issue.ID),
		nullablePositiveID(issue.ComicVineID),
		comicID,
	)
	if err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=metadata: %v", comicID, issue.ID, err)
		return nil, huma.Error500InternalServerError("failed to update comic from Metron")
	}
	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}
	if err := syncMetronIssueArcsWithOptions(ctx, db, client, comicID, issue, MetronImportOptions{Mode: "full"}); err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=arcs: %v", comicID, issue.ID, err)
		return nil, err
	}
	if err := syncMetronIssueCharacters(ctx, db, covers, comicID, issue); err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=characters: %v", comicID, issue.ID, err)
		return nil, err
	}
	output, err := getComic(ctx, db, comicID)
	if err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=response: %v", comicID, issue.ID, err)
		return nil, err
	}
	return output, nil
}

func rejectMetronIssueLinkedToAnotherComic(ctx context.Context, db *sqlx.DB, comicID, metronIssueID int) error {
	if metronIssueID <= 0 {
		return nil
	}
	var linked Comic
	err := db.GetContext(ctx, &linked, `
		SELECT id, series, series_year, issue
		FROM comics
		WHERE metron_issue_id = ? AND id <> ?
	`, metronIssueID, comicID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Printf("Metron comic update failed: comic_id=%d metron_issue_id=%d stage=conflict-check: %v", comicID, metronIssueID, err)
		return huma.Error500InternalServerError("failed to check whether the Metron issue is already linked")
	}
	log.Printf("Metron comic update rejected: comic_id=%d metron_issue_id=%d already_linked_comic_id=%d", comicID, metronIssueID, linked.ID)
	return metronIssueLinkConflict(metronIssueID, linked)
}

func mergeComicLinkedToMetronIssue(ctx context.Context, db *sqlx.DB, comicID, metronIssueID int) (*ComicDetailOutput, error) {
	if metronIssueID <= 0 {
		return nil, nil
	}
	var linkedID int
	err := db.GetContext(ctx, &linkedID, `
		SELECT id
		FROM comics
		WHERE metron_issue_id = ? AND id <> ?
	`, metronIssueID, comicID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to find the comic linked to this Metron issue")
	}
	return mergeComic(ctx, db, comicID, linkedID)
}

func metronIssueLinkConflict(metronIssueID int, linked Comic) error {
	return &huma.ErrorModel{
		Type:   metronIssueAlreadyLinkedProblem,
		Title:  http.StatusText(http.StatusConflict),
		Status: http.StatusConflict,
		Detail: fmt.Sprintf(
			"Metron issue %d is already linked to %s (comic %d). Merge the duplicate comics or choose another Metron issue.",
			metronIssueID,
			comicTitle(linked),
			linked.ID,
		),
	}
}

func ensureMetronIssueSeriesRow(ctx context.Context, db *sqlx.DB, issue metron.Issue) (int, error) {
	seriesID, err := ensureSeriesRow(ctx, db, issue.Series, issue.SeriesYear)
	if err != nil {
		return 0, err
	}
	if seriesID == 0 || issue.SeriesID <= 0 {
		return seriesID, nil
	}
	if err := updateSeriesMetronMetadata(ctx, db, seriesID, metron.Series{
		ID:        issue.SeriesID,
		Name:      issue.Series,
		YearBegan: issue.SeriesYear,
		Publisher: issue.Publisher,
	}); err != nil {
		return 0, err
	}
	return seriesID, nil
}
