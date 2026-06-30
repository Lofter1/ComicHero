package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Lofter1/ComicHero/backend/internal/api"
	"github.com/Lofter1/ComicHero/backend/internal/db"
	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

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
	router.Mount("/api", apiRouter)
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	humaAPI := humachi.New(apiRouter, api.DocsConfig())
	covers := api.NewCoverCache(env("COVER_CACHE_DIR", "./public/covers"), "/covers")
	if err := covers.EnsureDir(); err != nil {
		log.Fatalf("failed to prepare cover cache: %v", err)
	}

	api.RegisterReadingOrderRoutes(humaAPI, database)
	api.RegisterArcRoutes(humaAPI, database)
	api.RegisterComicRoutes(humaAPI, database, covers)
	metronClient := metron.New(metron.Config{
		BaseURL:  env("METRON_BASE_URL", metron.DefaultBaseURL),
		Username: os.Getenv("METRON_USERNAME"),
		Password: os.Getenv("METRON_PASSWORD"),
	})
	metronImportJobs := api.NewMetronImportJobStore()
	api.RegisterSeriesRoutes(humaAPI, database, metronClient, covers, metronImportJobs)
	api.RegisterCharacterRoutes(humaAPI, database)
	api.RegisterMetronRoutes(humaAPI, database, metronClient, covers, metronImportJobs)
	serveCovers(router, "/covers", covers.Dir())
	serveStatic(router, env("STATIC_DIR", "./public"))

	addr := ":" + env("PORT", "8080")
	log.Printf("listening on %s", addr)
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

func serveStatic(router chi.Router, dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		log.Printf("static dir %q not found; serving API only", dir)
		return
	}

	files := http.FileServer(http.Dir(dir))
	router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
		path := filepath.Join(dir, requestPath)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	}))
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
