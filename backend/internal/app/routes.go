package app

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/api"
	"github.com/Lofter1/ComicHero/backend/internal/config"
	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func registerRoutes(cfg config.Config, humaAPI huma.API, database *sqlx.DB, metronClient *metron.Client, covers *api.CoverCache) func() {
	importJobs := api.NewMetronImportJobStore()
	comicScanner := api.NewMetronComicScanner(database, metronClient, covers)
	comicDiscovery := api.NewMetronComicDiscovery(database, metronClient, covers)

	comicScanner.Start()
	comicDiscovery.Start()

	api.RegisterSystemRoutes(humaAPI, cfg.Version)
	api.RegisterReadingOrderRoutes(humaAPI, database, covers)
	api.RegisterUserRoutes(humaAPI, database)
	api.RegisterDashboardRoutes(humaAPI, database)
	api.RegisterStatisticsRoutes(humaAPI, database)
	api.RegisterArcRoutes(humaAPI, database)
	api.RegisterComicRoutes(humaAPI, database, covers)
	api.RegisterSeriesRoutes(humaAPI, database, metronClient, covers, importJobs)
	api.RegisterCharacterRoutes(humaAPI, database)
	api.RegisterMetronRoutes(humaAPI, database, metronClient, covers, importJobs, comicScanner)
	api.RegisterMetronComicDiscoveryRoutes(humaAPI, database, comicDiscovery)

	return func() {
		comicDiscovery.Stop()
		comicScanner.Stop()
	}
}
