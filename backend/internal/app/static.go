package app

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Lofter1/ComicHero/backend/internal/static"
)

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

// serveStatic serves a frontend directory when configured and otherwise falls
// back to the frontend embedded into the application binary.
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
			if noCacheStaticAsset(requestPath) {
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

func noCacheStaticAsset(path string) bool {
	switch path {
	case "sw.js", "registerSW.js", "manifest.webmanifest":
		return true
	default:
		return false
	}
}
