// Package app assembles ComicHero's database, API, background workers, and HTTP server.
package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Lofter1/ComicHero/backend/internal/api"
	"github.com/Lofter1/ComicHero/backend/internal/config"
	"github.com/Lofter1/ComicHero/backend/internal/db"
)

// Run starts ComicHero and blocks until its HTTP server exits.
func Run(cfg config.Config) error {
	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	covers := api.NewCoverCache(cfg.CoverCacheDir, cfg.CoverPublicPath)
	if err := covers.EnsureDir(); err != nil {
		return fmt.Errorf("prepare cover cache: %w", err)
	}

	handler, closeApplication, err := buildHandler(cfg, database, covers)
	if err != nil {
		return err
	}
	defer closeApplication()

	log.Printf("ComicHero %s listening on %s", cfg.Version, cfg.Address)
	if err := http.ListenAndServe(cfg.Address, handler); err != nil {
		return fmt.Errorf("server stopped: %w", err)
	}
	return nil
}
