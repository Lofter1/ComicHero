package api

import (
	"context"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func registerMetronCharactersRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
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
}
