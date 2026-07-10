package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Lofter1/ComicHero/backend/internal/api"
	"github.com/Lofter1/ComicHero/backend/internal/db"
	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/Lofter1/ComicHero/backend/internal/static"
)

var version = "dev"

func main() {
	loadEnvFiles(".env", "../.env")

	database, err := db.Open(env("DB_PATH", "./data/comicorder.db"))
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	apiRouter := chi.NewRouter()
	apiRouter.Use(api.UserMiddleware(database))
	router.Mount("/api", apiRouter)
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	humaAPI := humachi.New(apiRouter, api.DocsConfig())
	covers := api.NewCoverCache(env("COVER_CACHE_DIR", "./public/covers"), "/covers")
	if err := covers.EnsureDir(); err != nil {
		log.Fatalf("failed to prepare cover cache: %v", err)
	}

	api.RegisterReadingOrderRoutes(humaAPI, database, covers)
	api.RegisterUserRoutes(humaAPI, database)
	api.RegisterDashboardRoutes(humaAPI, database)
	api.RegisterStatisticsRoutes(humaAPI, database)
	api.RegisterArcRoutes(humaAPI, database)
	api.RegisterComicRoutes(humaAPI, database, covers)
	metronClient := metron.New(metron.Config{
		BaseURL:  env("METRON_BASE_URL", metron.DefaultBaseURL),
		Username: os.Getenv("METRON_USERNAME"),
		Password: os.Getenv("METRON_PASSWORD"),
	})
	metronImportJobs := api.NewMetronImportJobStore()
	metronComicScanner := api.NewMetronComicScanner(database, metronClient, covers)
	metronComicScanner.Start()
	defer metronComicScanner.Stop()
	metronComicDiscovery := api.NewMetronComicDiscovery(database, metronClient, covers)
	metronComicDiscovery.Start()
	defer metronComicDiscovery.Stop()
	api.RegisterSeriesRoutes(humaAPI, database, metronClient, covers, metronImportJobs)
	api.RegisterCharacterRoutes(humaAPI, database)
	api.RegisterMetronRoutes(humaAPI, database, metronClient, covers, metronImportJobs, metronComicScanner)
	api.RegisterMetronComicDiscoveryRoutes(humaAPI, database, metronComicDiscovery)
	serveCovers(router, "/covers", covers.Dir())
	if err := serveStatic(router, os.Getenv("STATIC_DIR")); err != nil {
		log.Fatalf("failed to prepare static assets: %v", err)
	}

	addr := ":" + env("PORT", "8080")
	log.Printf("ComicHero %s listening on %s", version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func serveCovers(router chi.Router, publicPath, dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		log.Printf("cover cache dir %q not found; covers unavailable", dir)
		return
	}

	prefix := "/" + strings.Trim(strings.TrimSpace(publicPath), "/")
	files := http.StripPrefix(prefix+"/", http.FileServer(http.Dir(dir)))
	router.Handle(prefix+"/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		files.ServeHTTP(w, r)
	}))
}

// serveStatic serves the frontend. If diskDir is non-empty and exists, it's
// served straight off disk (handy for local frontend dev). Otherwise it
// falls back to the frontend embedded into the binary at build time.
func serveStatic(router chi.Router, diskDir string) error {
	var fsys fs.FS

	if diskDir != "" {
		if info, err := os.Stat(diskDir); err == nil && info.IsDir() {
			log.Printf("serving frontend from disk: %s", diskDir)
			fsys = os.DirFS(diskDir)
		} else {
			log.Printf("STATIC_DIR %q not found; falling back to embedded frontend", diskDir)
		}
	}

	if fsys == nil {
		embedded, err := static.FS()
		if err != nil {
			return fmt.Errorf("embedded frontend: %w", err)
		}
		fsys = embedded
	}

	files := http.FileServer(http.FS(fsys))
	router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
		if info, err := fs.Stat(fsys, requestPath); err == nil && !info.IsDir() {
			if requestPath == "sw.js" || requestPath == "registerSW.js" || requestPath == "manifest.webmanifest" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
			return
		}

		index, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
	}))
	return nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func loadEnvFiles(paths ...string) {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			setEnvLine(scanner.Text())
		}
		if err := file.Close(); err != nil {
			log.Printf("failed to close env file %q: %v", path, err)
		}
	}
}

func setEnvLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	line = strings.TrimPrefix(line, "export ")
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return
	}

	key = strings.TrimSpace(key)
	if key == "" || os.Getenv(key) != "" {
		return
	}

	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	os.Setenv(key, value)
}
