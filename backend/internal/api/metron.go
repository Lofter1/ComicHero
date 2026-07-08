package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func RegisterMetronRoutes(api huma.API, db *sqlx.DB, client *metron.Client, covers *CoverCache, importJobs *metronImportJobStore) {

	registerMetronComicRoutes(api, db, client, covers, importJobs)

	registerMetronReadingOrdersRoutes(api, db, client, covers, importJobs)

	registerMetronArcsRoutes(api, db, client, covers, importJobs)

	registerMetronSeriesRoutes(api, db, client, covers, importJobs)

	registerMetronCharactersRoutes(api, db, client, covers, importJobs)

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

	registerMetronJobRoutes(api, db, client, covers, importJobs)

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
}
