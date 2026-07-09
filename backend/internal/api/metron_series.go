package api

import (
	"context"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

func registerMetronSeriesRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {
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
}

func importMetronSeries(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue) (*ComicListOutput, error) {
	return importMetronSeriesWithProgress(ctx, db, client, covers, issues, func(int, int, string) {})
}

func importMetronSeriesWithProgress(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue, progress func(int, int, string)) (*ComicListOutput, error) {
	return importMetronSeriesWithProgressOptions(ctx, db, client, covers, issues, progress, defaultMetronImportOptions())
}

func importMetronSeriesWithProgressOptions(ctx context.Context, db *sqlx.DB, client *metron.Client, covers *CoverCache, issues []metron.Issue, progress func(int, int, string), options MetronImportOptions) (*ComicListOutput, error) {
	options = resolveMetronImportOptions(options)
	comics := make([]Comic, 0, len(issues))
	total := len(issues)
	progress(0, total, "Importing series issues...")
	for i, issue := range issues {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if options.Mode != "full" && !options.Force && issue.ID > 0 {
			if id, ok, err := existingComicIDByMetronIssueID(ctx, db, issue.ID); err != nil {
				return nil, err
			} else if ok {
				if err := linkComicToMetronIssueSeries(ctx, db, id, issue); err != nil {
					return nil, err
				}
				comic, err := getComicRow(ctx, db, id)
				if err != nil {
					return nil, err
				}
				comics = append(comics, comic)
				progress(i+1, total, "Importing series issues...")
				continue
			}
		}

		if options.Mode != "full" && !options.Force {
			if id, ok, err := existingComicIDByMetronIssueMatch(ctx, db, issue); err != nil {
				return nil, err
			} else if ok {
				if issue.ID > 0 {
					if err := attachMetronIssueID(ctx, db, id, issue.ID); err != nil {
						return nil, err
					}
				}
				if err := linkComicToMetronIssueSeries(ctx, db, id, issue); err != nil {
					return nil, err
				}
				comic, err := getComicRow(ctx, db, id)
				if err != nil {
					return nil, err
				}
				comics = append(comics, comic)
				progress(i+1, total, "Importing series issues...")
				continue
			}
		}

		comic, err := importMetronComicSweep(ctx, db, client, covers, issue, options, true)
		if err != nil {
			return nil, err
		}
		comics = append(comics, comic.Body.Comic)
		progress(i+1, total, "Importing series issues...")
	}
	progress(total, total, "Series imported.")
	return &ComicListOutput{Body: comics}, nil
}
