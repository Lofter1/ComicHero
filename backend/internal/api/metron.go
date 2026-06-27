package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"

	"ComicHeroV2-Backend/internal/metron"
)

type MetronIssueListInput struct {
	Query  string `query:"q"      doc:"Search text for Metron issues. Used as series-name search when series is empty." example:"Batman"`
	Series string `query:"series" doc:"Metron series-name filter." example:"Batman"`
	Issue  int    `query:"issue"  doc:"Numeric issue-number filter." minimum:"1" example:"6"`
}

type MetronIDInput struct {
	ID int `path:"id" doc:"Metron resource identifier." minimum:"1" example:"123456"`
}

type MetronReadingListInput struct {
	Query string `query:"q" doc:"Search text for Metron reading lists." example:"Court of Owls"`
}

type MetronSeriesInput struct {
	Query string `query:"q" doc:"Search text for Metron series." example:"Batman"`
}

type MetronIssueListOutput struct {
	MetronRateLimitHeaders
	Body []metron.Issue
}

type MetronIssueOutput struct {
	MetronRateLimitHeaders
	Body metron.Issue
}

type MetronReadingListOutput struct {
	MetronRateLimitHeaders
	Body []metron.ReadingList
}

type MetronReadingListDetailOutput struct {
	MetronRateLimitHeaders
	Body metron.ReadingList
}

type MetronSeriesListOutput struct {
	MetronRateLimitHeaders
	Body []metron.Series
}

type MetronRateLimitHeaders struct {
	BurstLimit         string `header:"X-RateLimit-Burst-Limit"         doc:"Metron burst-rate request limit, forwarded from the latest Metron response."`
	BurstRemaining     string `header:"X-RateLimit-Burst-Remaining"     doc:"Remaining Metron burst-rate requests, forwarded from the latest Metron response."`
	BurstReset         string `header:"X-RateLimit-Burst-Reset"         doc:"Unix timestamp when the Metron burst-rate window resets."`
	SustainedLimit     string `header:"X-RateLimit-Sustained-Limit"     doc:"Metron sustained-rate request limit, forwarded from the latest Metron response."`
	SustainedRemaining string `header:"X-RateLimit-Sustained-Remaining" doc:"Remaining Metron sustained-rate requests, forwarded from the latest Metron response."`
	SustainedReset     string `header:"X-RateLimit-Sustained-Reset"     doc:"Unix timestamp when the Metron sustained-rate window resets."`
}

func RegisterMetronRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
	huma.Register(api, huma.Operation{
		OperationID: "searchMetronComics",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron comics",
		Description: "Searches Metron for comic issue metadata. When series is omitted, q is sent as the Metron series-name search.",
		Method:      http.MethodGet,
		Path:        "/metron/comics",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIssueListInput) (*MetronIssueListOutput, error) {
		issues, err := client.SearchIssues(ctx, input.Query, input.Series, input.Issue)
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
	}, func(ctx context.Context, input *MetronIDInput) (*MetronImportJobOutput, error) {
		job := startMetronComicImport(importJobs, db, client, covers, input.ID)
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
		issue, err := client.GetIssue(ctx, input.Body.MetronIssueID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		output, err := updateComicFromMetron(ctx, db, covers, input.ID, *issue)
		if err != nil {
			return nil, err
		}
		return withMetronRateLimit(output, client.CurrentRateLimit()), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronReadingLists",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron reading lists",
		Description: "Searches Metron for reading lists.",
		Method:      http.MethodGet,
		Path:        "/metron/readingLists",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronReadingListInput) (*MetronReadingListOutput, error) {
		lists, err := client.SearchReadingLists(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronReadingListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: lists}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronReadingList",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron reading list",
		Description: "Gets a Metron reading list by ID, including its issue entries when Metron returns them.",
		Method:      http.MethodGet,
		Path:        "/metron/readingLists/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronReadingListDetailOutput, error) {
		list, err := client.GetReadingList(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronReadingListDetailOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *list}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronReadingList",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron reading list",
		Description:   "Starts a background job that imports a Metron reading list as a local reading order and imports its issues as local comics. If the reading list is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/readingLists/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronImportJobOutput, error) {
		job := startMetronReadingListImport(importJobs, db, client, covers, input.ID)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronSeries",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron series",
		Description: "Searches Metron for comic series.",
		Method:      http.MethodGet,
		Path:        "/metron/series",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronSeriesInput) (*MetronSeriesListOutput, error) {
		series, err := client.SearchSeries(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronSeriesListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: series}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronSeries",
		Tags:          []string{tagMetron},
		Summary:       "Import Metron series",
		Description:   "Starts a background job that imports every missing issue in a Metron series for use in reading orders. Already imported issues are skipped before detail calls are made.",
		Method:        http.MethodPost,
		Path:          "/metron/series/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronImportJobOutput, error) {
		job := startMetronSeriesImport(importJobs, db, client, covers, input.ID)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron import job",
		Description: "Returns the current status of a background Metron import job.",
		Method:      http.MethodGet,
		Path:        "/metron/imports/{id}",
		Errors:      errsRead,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return getMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "cancelMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Cancel Metron import job",
		Description: "Requests cancellation for a queued or running background Metron import job.",
		Method:      http.MethodPost,
		Path:        "/metron/imports/{id}/cancel",
		Errors:      errsRead,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return cancelMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID:   "continueMetronImportJob",
		Tags:          []string{tagMetron},
		Summary:       "Continue Metron import job",
		Description:   "Starts a new background import for the same Metron resource as a canceled import job.",
		Method:        http.MethodPost,
		Path:          "/metron/imports/{id}/continue",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportJobInput) (*MetronImportJobOutput, error) {
		return continueMetronImportJob(importJobs, db, client, covers, input.ID)
	})
}

func metronAPIError(err error) error {
	var rateLimitErr *metron.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return huma.NewError(http.StatusTooManyRequests, rateLimitErr.Error())
	}
	return huma.Error502BadGateway(err.Error())
}

func metronRateLimitHeaders(rateLimit metron.RateLimit) MetronRateLimitHeaders {
	if rateLimit.Empty() {
		return MetronRateLimitHeaders{}
	}
	return MetronRateLimitHeaders{
		BurstLimit:         strconv.Itoa(rateLimit.BurstLimit),
		BurstRemaining:     strconv.Itoa(rateLimit.BurstRemaining),
		BurstReset:         strconv.FormatInt(rateLimit.BurstReset, 10),
		SustainedLimit:     strconv.Itoa(rateLimit.SustainedLimit),
		SustainedRemaining: strconv.Itoa(rateLimit.SustainedRemaining),
		SustainedReset:     strconv.FormatInt(rateLimit.SustainedReset, 10),
	}
}

func withMetronRateLimit[T interface {
	*ComicDetailOutput | *ReadingOrderDetailOutput | *ComicListOutput
}](output T, rateLimit metron.RateLimit) T {
	headers := metronRateLimitHeaders(rateLimit)
	switch typed := any(output).(type) {
	case *ComicDetailOutput:
		typed.MetronRateLimitHeaders = headers
	case *ReadingOrderDetailOutput:
		typed.MetronRateLimitHeaders = headers
	case *ComicListOutput:
		typed.MetronRateLimitHeaders = headers
	}
	return output
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

func importMetronComic(ctx context.Context, db *sqlx.DB, covers *CoverCache, issue metron.Issue) (*ComicDetailOutput, error) {
	if issue.ID > 0 {
		if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
			return nil, err
		} else if ok {
			return getComic(ctx, db, id)
		}
	}

	if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
		return nil, err
	} else if ok {
		if issue.ID > 0 {
			if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
				return nil, err
			}
		}
		return getComic(ctx, db, id)
	}

	return createMetronComic(ctx, db, covers, issue)
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

func attachMetronIssueID(ctx context.Context, db *sqlx.DB, comicID, metronID int) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE comics SET metron_issue_id = ? WHERE id = ? AND metron_issue_id IS NULL
	`, metronID, comicID); err != nil {
		return huma.Error500InternalServerError("failed to link comic to Metron")
	}
	return nil
}

func createMetronComic(ctx context.Context, db *sqlx.DB, covers *CoverCache, issue metron.Issue) (*ComicDetailOutput, error) {
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, read, metron_issue_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		payload.Read,
		nullableMetronID(issue.ID),
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron comic")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported comic id")
	}
	return getComic(ctx, db, int(id))
}

func updateComicFromMetron(ctx context.Context, db *sqlx.DB, covers *CoverCache, comicID int, issue metron.Issue) (*ComicDetailOutput, error) {
	payload := comicPayloadFromMetronIssue(issue)
	var err error
	payload.CoverImage, err = localCoverURL(ctx, covers, payload.CoverImage)
	if err != nil {
		return nil, err
	}

	result, err := db.ExecContext(ctx, `
		UPDATE comics
		SET series = ?, series_year = ?, issue = ?, publisher = ?, cover_date = ?, cover_image = ?, description = ?, metron_issue_id = ?
		WHERE id = ?
	`, payload.Series,
		payload.SeriesYear,
		payload.Issue,
		payload.Publisher,
		payload.CoverDate,
		payload.CoverImage,
		payload.Description,
		nullableMetronID(issue.ID),
		comicID,
	)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to update comic from Metron")
	}
	if err := requireRowsAffected(result, "comic not found"); err != nil {
		return nil, err
	}

	return getComic(ctx, db, comicID)
}

func importMetronReadingList(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList) (*ReadingOrderDetailOutput, error) {
	if list.ID > 0 {
		if id, ok, err := existingReadingOrderIDByMetronID(ctx, db, list.ID); err != nil {
			return nil, err
		} else if ok {
			return getReadingOrder(ctx, db, id)
		}
	}

	order, err := createMetronReadingOrder(ctx, db, list)
	if err != nil {
		return nil, err
	}

	input := &SetReadingOrderComicsInput{ID: order.Body.ID}
	for _, issue := range list.Issues {
		comic, err := importMetronComic(ctx, db, covers, issue)
		if err != nil {
			return nil, err
		}
		input.Body.Comics = append(input.Body.Comics, ReadingOrderComicPayload{
			ComicID: comic.Body.ID,
		})
	}

	return setReadingOrderComics(ctx, db, input)
}

func importMetronReadingListWithProgress(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList, progress func(int, int, string)) error {
	return importMetronReadingListWithOptions(ctx, db, covers, list, false, progress)
}

func continueMetronReadingListWithProgress(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList, progress func(int, int, string)) error {
	return importMetronReadingListWithOptions(ctx, db, covers, list, true, progress)
}

func importMetronReadingListWithOptions(ctx context.Context, db *sqlx.DB, covers *CoverCache, list metron.ReadingList, continueExisting bool, progress func(int, int, string)) error {
	var orderID int
	if list.ID > 0 {
		if id, ok, err := existingReadingOrderIDByMetronID(ctx, db, list.ID); err != nil || ok {
			if ok {
				if !continueExisting {
					progress(1, 1, "Reading list already exists.")
					return err
				}
				orderID = id
			}
			if err != nil {
				return err
			}
		}
	}

	if orderID == 0 {
		order, err := createMetronReadingOrder(ctx, db, list)
		if err != nil {
			return err
		}
		orderID = order.Body.ID
	}

	input := &SetReadingOrderComicsInput{ID: orderID}
	total := len(list.Issues)
	progress(0, total, "Importing reading-list issues...")
	for i, issue := range list.Issues {
		if err := ctx.Err(); err != nil {
			return err
		}
		comic, err := importMetronComic(ctx, db, covers, issue)
		if err != nil {
			return err
		}
		input.Body.Comics = append(input.Body.Comics, ReadingOrderComicPayload{
			ComicID: comic.Body.ID,
		})
		progress(i+1, total, "Importing reading-list issues...")
	}

	if _, err := setReadingOrderComics(ctx, db, input); err != nil {
		return err
	}
	progress(total, total, "Reading list imported.")
	return nil
}

func importMetronSeries(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue) (*ComicListOutput, error) {
	return importMetronSeriesWithProgress(ctx, db, client, covers, issues, func(int, int, string) {})
}

func importMetronSeriesWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue, progress func(int, int, string)) (*ComicListOutput, error) {
	comics := make([]Comic, 0, len(issues))
	total := len(issues)
	progress(0, total, "Importing series issues...")
	for i, issue := range issues {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if issue.ID > 0 {
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				comic, err := getComic(ctx, db, id)
				if err != nil {
					return nil, err
				}
				comics = append(comics, comic.Body.Comic)
				progress(i+1, total, "Importing series issues...")
				continue
			}
		}

		if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
			return nil, err
		} else if ok {
			if issue.ID > 0 {
				if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
					return nil, err
				}
			}
			comic, err := getComic(ctx, db, id)
			if err != nil {
				return nil, err
			}
			comics = append(comics, comic.Body.Comic)
			progress(i+1, total, "Importing series issues...")
			continue
		}

		fullIssue := issue
		if issue.ID > 0 {
			detail, err := client.GetIssue(ctx, issue.ID)
			if err != nil {
				if isContextCanceledError(err) {
					return nil, err
				}
				return nil, huma.Error502BadGateway(err.Error())
			}
			fullIssue = *detail
		}

		comic, err := importMetronComic(ctx, db, covers, fullIssue)
		if err != nil {
			return nil, err
		}
		comics = append(comics, comic.Body.Comic)
		progress(i+1, total, "Importing series issues...")
	}
	progress(total, total, "Series imported.")
	return &ComicListOutput{Body: comics}, nil
}

func existingReadingOrderIDByMetronID(ctx context.Context, db *sqlx.DB, metronID int) (int, bool, error) {
	var id int
	if err := db.GetContext(ctx, &id, `
		SELECT id FROM reading_orders WHERE metron_reading_list_id = ?
	`, metronID); err != nil {
		if err != sql.ErrNoRows {
			return 0, false, huma.Error500InternalServerError("failed to check imported reading list")
		}
		return 0, false, nil
	}
	return id, true, nil
}

func createMetronReadingOrder(ctx context.Context, db *sqlx.DB, list metron.ReadingList) (*CreateReadingOrderOutput, error) {
	result, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (name, description, favorite, metron_reading_list_id)
		VALUES (?, ?, ?, ?)
	`, list.Name, list.Description, false, nullableMetronID(list.ID))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to import Metron reading list")
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to get imported reading order id")
	}

	var ro ReadingOrder
	if err := db.GetContext(ctx, &ro, `
		SELECT * FROM reading_orders WHERE id = ?
	`, id); err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch imported reading order")
	}

	return &CreateReadingOrderOutput{Body: ro}, nil
}

func nullableMetronID(id int) any {
	if id <= 0 {
		return nil
	}
	return id
}
