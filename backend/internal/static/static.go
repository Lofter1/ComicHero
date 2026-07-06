// Package static embeds the built frontend (ui/dist) into the compiled
// binary so the server can run standalone with no external asset directory.
//
// The dist/ subdirectory here is populated by the build (see Makefile /
// Dockerfile: `npm run build` in ui/, then copy ui/dist -> this dist/
// before `go build`). It is gitignored and empty in source control.
package static

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// FS returns the embedded frontend files rooted at dist/, ready to be
// served directly (e.g. via http.FS).
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "dist")
}
