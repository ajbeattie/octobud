// Package web embeds the frontend static files for production deployment.
//
// The dist/ directory is populated by running `npm run build` in the frontend
// directory, then copying the output to backend/web/dist/.
//
// Use the build-frontend target: make build-frontend
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns a filesystem containing the built frontend assets.
// The assets are rooted at "dist/", so callers typically use fs.Sub to strip that prefix.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
