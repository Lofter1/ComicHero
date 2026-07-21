package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"
)

type CBLRepositorySyncStatusOutput struct{ Body CBLRepositorySyncStatus }
type UpdateCBLRepositorySyncSettingsInput struct{ Body CBLRepositorySyncSettings }
type CBLRepositoryFilesOutput struct{ Body []CBLRepositoryFile }
type TriggerCBLRepositorySyncInput struct {
	Body struct {
		Files                  []CBLRepositoryFileSelection `json:"files,omitempty"`
		ResolveMissingOnMetron bool                         `json:"resolveMissingOnMetron,omitempty"`
	}
}
type ResolveCBLMetronIssueInput struct {
	Body struct {
		ResolutionID  string `json:"resolutionId" minLength:"1"`
		MetronIssueID int    `json:"metronIssueId" minimum:"0"`
	}
}

func RegisterCBLRepositorySyncRoutes(api huma.API, db *sqlx.DB, syncer *cblRepositorySyncer) {
	huma.Register(api, huma.Operation{OperationID: "getCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Get CBL repository sync settings and status", Method: http.MethodGet, Path: "/readingOrders/repository-sync", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Stream CBL repository sync status", Method: http.MethodGet, Path: "/readingOrders/repository-sync/events", Errors: []int{401, 403, 500}}, map[string]any{"sync": CBLRepositorySyncEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		updates, unsubscribe := syncer.subscribe(ctx)
		defer unsubscribe()
		for {
			select {
			case <-ctx.Done():
				return
			case status, ok := <-updates:
				if !ok || send.Data(CBLRepositorySyncEvent{Sync: status}) != nil {
					return
				}
			}
		}
	})
	huma.Register(api, huma.Operation{OperationID: "updateCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Update CBL repository sync settings", Method: http.MethodPut, Path: "/readingOrders/repository-sync", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateCBLRepositorySyncSettingsInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateCBLRepositorySyncSettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveCBLRepositorySyncSettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save CBL repository settings")
		}
		select {
		case syncer.wake <- struct{}{}:
		default:
		}
		syncer.broadcast()
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "listCBLRepositoryFiles", Tags: []string{tagReadingOrders}, Summary: "List available repository CBL files", Method: http.MethodGet, Path: "/readingOrders/repository-sync/files", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositoryFilesOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		settings, err := loadCBLRepositorySyncSettings(ctx, db)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load CBL repository settings")
		}
		files, err := syncer.availableRepositoryFiles(ctx, settings)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &CBLRepositoryFilesOutput{Body: files}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Import all or selected CBL repository files", Method: http.MethodPost, Path: "/readingOrders/repository-sync/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, input *TriggerCBLRepositorySyncInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := syncer.trigger("manual", input.Body.Files, input.Body.ResolveMissingOnMetron); err != nil {
			if strings.Contains(err.Error(), "already running") {
				return nil, huma.Error409Conflict(err.Error())
			}
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "resolveCBLRepositoryMetronIssue", Tags: []string{tagReadingOrders}, Summary: "Choose a Metron issue for an ambiguous CBL book", Method: http.MethodPost, Path: "/readingOrders/repository-sync/resolve", Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, input *ResolveCBLMetronIssueInput) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := syncer.resolveMetronIssue(strings.TrimSpace(input.Body.ResolutionID), input.Body.MetronIssueID); err != nil {
			return nil, huma.Error409Conflict(err.Error())
		}
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "stopCBLRepositorySync", Tags: []string{tagReadingOrders}, Summary: "Stop CBL repository import", Method: http.MethodPost, Path: "/readingOrders/repository-sync/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*CBLRepositorySyncStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		syncer.stop("stopped by admin")
		return &CBLRepositorySyncStatusOutput{Body: syncer.snapshot(ctx)}, nil
	})
}
