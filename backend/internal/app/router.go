package app

import (
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"github.com/Lofter1/ComicHero/backend/internal/api"
	"github.com/Lofter1/ComicHero/backend/internal/config"
	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func buildHandler(cfg config.Config, database *sqlx.DB, covers *api.CoverCache) (http.Handler, func(), error) {
	accessLog, err := newAccessLogger(cfg.AccessLogPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open access log: %w", err)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(accessLog.Middleware)

	apiRouter := chi.NewRouter()
	apiRouter.Use(api.UserMiddleware(database))
	apiRouter.Use(api.AuditMiddleware(database))
	router.Mount("/api", apiRouter)
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	humaAPI := humachi.New(apiRouter, api.DocsConfig())
	metronClient := metron.New(metron.Config{
		BaseURL:  cfg.MetronBaseURL,
		Username: cfg.MetronUsername,
		Password: cfg.MetronPassword,
	})
	stopWorkers := registerRoutes(cfg, humaAPI, database, metronClient, covers)

	serveCovers(router, cfg.CoverPublicPath, covers.Dir())
	if err := serveStatic(router, cfg.StaticDir); err != nil {
		stopWorkers()
		_ = accessLog.Close()
		return nil, nil, fmt.Errorf("prepare static assets: %w", err)
	}

	cleanup := func() {
		stopWorkers()
		_ = accessLog.Close()
	}
	return router, cleanup, nil
}
