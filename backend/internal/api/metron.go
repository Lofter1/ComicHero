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

	registerMetronComicRoutes(api, db, client, covers, importJobs)

	registerMetronReadingOrdersRoutes(api, db, client, covers, importJobs)

	registerMetronArcsRoutes(api, db, client, covers, importJobs)

	registerMetronSeriesRoutes(api, db, client, covers, importJobs)

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
