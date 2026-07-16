package api

import (
	"context"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"
)

type MetronImportJob struct {
	ID        string              `json:"id" doc:"Import job identifier." example:"metron-1"`
	Type      string              `json:"type" doc:"Import type." enum:"comic,readingList,readingLists,series,character,arc" example:"series"`
	MetronID  int                 `json:"metronId" doc:"Metron resource identifier." example:"123456"`
	Options   MetronImportOptions `json:"options" doc:"Import depth and data-expansion options used by this job."`
	Status    string              `json:"status" doc:"Current job status." enum:"queued,running,canceling,succeeded,failed,canceled" example:"running"`
	Message   string              `json:"message" doc:"Human-readable status message." example:"Importing series from Metron..."`
	Completed int                 `json:"completed" doc:"Completed import units." example:"12"`
	Total     int                 `json:"total" doc:"Total import units, when known." example:"52"`
	StartedAt string              `json:"startedAt" doc:"RFC3339 start timestamp." example:"2026-06-26T12:30:00Z"`
	EndedAt   string              `json:"endedAt,omitempty" doc:"RFC3339 end timestamp, when finished." example:"2026-06-26T12:31:00Z"`
}

type MetronImportJobOutput struct {
	Body MetronImportJob
}

type MetronImportJobListOutput struct {
	Body []MetronImportJob
}

type MetronImportJobEvent struct {
	Job MetronImportJob `json:"job" doc:"Updated Metron import job."`
}

type MetronImportJobInput struct {
	ID string `path:"id" doc:"Import job identifier." example:"metron-1"`
}

func registerMetronJobRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
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
