package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

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
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/comics"); err != nil {
			return nil, err
		}
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
		issue, info, err := fetchMetronIssue(ctx, db, client, input.Body.MetronIssueID, input.Body.Force)
		if err != nil {
			return nil, metronAPIError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, input.Body.MetronIssueID); err != nil {
				return nil, err
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

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronReadingLists",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron reading lists",
		Description: "Searches Metron for reading lists.",
		Method:      http.MethodGet,
		Path:        "/metron/readingLists",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronReadingListInput) (*MetronReadingListOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/readingLists"); err != nil {
			return nil, err
		}
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
		if err := authorizeMetron(ctx, db, metronScopeDetail, "GET /metron/readingLists/{id}"); err != nil {
			return nil, err
		}
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
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/readingLists/{id}/import"); err != nil {
			return nil, err
		}
		job := startMetronReadingListImportWithOptions(ctx, importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronArcs",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron arcs",
		Description: "Searches Metron for story arcs.",
		Method:      http.MethodGet,
		Path:        "/metron/arcs",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronArcInput) (*MetronArcListOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/arcs"); err != nil {
			return nil, err
		}
		arcs, err := client.SearchArcs(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronArcListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: arcs}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronArc",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron arc",
		Description: "Gets a Metron story arc by ID, including its issue entries.",
		Method:      http.MethodGet,
		Path:        "/metron/arcs/{id}",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronIDInput) (*MetronArcDetailOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeDetail, "GET /metron/arcs/{id}"); err != nil {
			return nil, err
		}
		arc, err := client.GetArc(ctx, input.ID)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronArcDetailOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: *arc}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronArc",
		Tags:          []string{tagMetron, tagArcs},
		Summary:       "Import Metron arc",
		Description:   "Starts a background job that imports a Metron story arc as a local arc and imports or reuses its issues as local comics. If the arc is already imported, the job finishes without calling Metron again.",
		Method:        http.MethodPost,
		Path:          "/metron/arcs/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/arcs/{id}/import"); err != nil {
			return nil, err
		}
		job := startMetronArcImportWithOptions(ctx, importJobs, db, client, covers, input.ID, input.Body)
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
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/series"); err != nil {
			return nil, err
		}
		series, err := client.SearchSeries(ctx, metron.SeriesSearchOptions{
			Query:     input.Query,
			YearBegan: input.YearBegan,
			Volume:    input.Volume,
		})
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronSeriesListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: series}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "searchMetronCharacters",
		Tags:        []string{tagMetron},
		Summary:     "Search Metron characters",
		Description: "Searches Metron for characters by name.",
		Method:      http.MethodGet,
		Path:        "/metron/characters",
		Errors:      errsMetronRead,
	}, func(ctx context.Context, input *MetronCharacterInput) (*MetronCharacterListOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeSearch, "GET /metron/characters"); err != nil {
			return nil, err
		}
		characters, err := client.SearchCharacters(ctx, input.Query)
		if err != nil {
			return nil, metronAPIError(err)
		}
		return &MetronCharacterListOutput{MetronRateLimitHeaders: metronRateLimitHeaders(client.CurrentRateLimit()), Body: characters}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "importMetronCharacterAppearances",
		Tags:          []string{tagMetron, tagCharacters},
		Summary:       "Import Metron character appearances",
		Description:   "Starts a background job that imports or reuses a Metron character locally, fetches the character's Metron issue list, imports or reuses those issues, and links them as local appearances.",
		Method:        http.MethodPost,
		Path:          "/metron/characters/{id}/import",
		DefaultStatus: http.StatusAccepted,
		Errors:        errsMetronSync,
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/characters/{id}/import"); err != nil {
			return nil, err
		}
		job := startMetronCharacterAppearancesImportWithOptions(ctx, importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
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
	}, func(ctx context.Context, input *MetronImportInput) (*MetronImportJobOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/series/{id}/import"); err != nil {
			return nil, err
		}
		job := startMetronSeriesImportWithOptions(ctx, importJobs, db, client, covers, input.ID, input.Body)
		return &MetronImportJobOutput{Body: job}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getMetronQuota",
		Tags:        []string{tagMetron},
		Summary:     "Get Metron quota",
		Description: "Returns the latest Metron rate-limit quota known to this server.",
		Method:      http.MethodGet,
		Path:        "/metron/quota",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronQuotaOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "GET /metron/quota"); err != nil {
			return nil, err
		}
		rateLimit := client.CurrentRateLimit()
		return &MetronQuotaOutput{
			MetronRateLimitHeaders: metronRateLimitHeaders(rateLimit),
			Body:                   metronQuotaFromRateLimit(rateLimit),
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listMetronImportJobs",
		Tags:        []string{tagMetron},
		Summary:     "List Metron import jobs",
		Description: "Returns background Metron import jobs so the web app can reconnect after a reload.",
		Method:      http.MethodGet,
		Path:        "/metron/imports",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronImportJobListOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "GET /metron/imports"); err != nil {
			return nil, err
		}
		return listMetronImportJobs(importJobs), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listMetronRequests",
		Tags:        []string{tagMetron},
		Summary:     "List recent Metron requests",
		Description: "Returns recent outbound Metron API calls recorded by this server, including path, query, status, duration, and conditional-request state.",
		Method:      http.MethodGet,
		Path:        "/metron/requests",
		Errors:      errsRead,
	}, func(ctx context.Context, input *struct{}) (*MetronRequestLogOutput, error) {
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "GET /metron/requests"); err != nil {
			return nil, err
		}
		return &MetronRequestLogOutput{Body: client.RecentRequests()}, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "streamMetronImportJobs",
		Tags:        []string{tagMetron},
		Summary:     "Stream Metron import jobs",
		Description: "Streams background Metron import job updates so the web app can reconnect after a reload without polling.",
		Method:      http.MethodGet,
		Path:        "/metron/imports/events",
		Errors:      errsRead,
	}, map[string]any{
		"job": MetronImportJobEvent{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "GET /metron/imports/events"); err != nil {
			return
		}
		streamMetronImportJobs(ctx, importJobs, func(event MetronImportJobEvent) error {
			return send.Data(event)
		})
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
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "GET /metron/imports/{id}"); err != nil {
			return nil, err
		}
		return getMetronImportJob(importJobs, input.ID)
	})

	huma.Register(api, huma.Operation{
		OperationID: "dismissMetronImportJob",
		Tags:        []string{tagMetron},
		Summary:     "Dismiss Metron import job",
		Description: "Removes a finished Metron import job from the monitor.",
		Method:      http.MethodDelete,
		Path:        "/metron/imports/{id}",
		Errors:      errsWrite,
	}, func(ctx context.Context, input *MetronImportJobInput) (*struct{}, error) {
		if err := authorizeMetron(ctx, db, metronScopeMonitor, "DELETE /metron/imports/{id}"); err != nil {
			return nil, err
		}
		return deleteMetronImportJob(importJobs, input.ID)
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
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/imports/{id}/cancel"); err != nil {
			return nil, err
		}
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
		if err := authorizeMetron(ctx, db, metronScopeImport, "POST /metron/imports/{id}/continue"); err != nil {
			return nil, err
		}
		return continueMetronImportJob(ctx, importJobs, db, client, covers, input.ID)
	})
}
