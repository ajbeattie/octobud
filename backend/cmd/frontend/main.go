// Copyright (C) 2025 Austin Beattie
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Package main provides a static file server for the embedded frontend.
// It serves the built SvelteKit app and proxies /api requests to the backend.
//
// This is useful for production deployments where you want a single binary
// to serve the frontend, while the API runs as a separate process.
//
// Build with: make build-frontend
//
// Usage:
//
//	./octobud-web                           # Serves on :3000, proxies to localhost:8080
//	./octobud-web -addr :5000               # Serves on :5000
//	./octobud-web -api http://api:8080      # Proxies to different backend
package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ajbeattie/octobud/backend/internal/api/shared"
	"github.com/ajbeattie/octobud/backend/web"
)

func main() {
	addr := flag.String("addr", ":3000", "address to listen on")
	apiURL := flag.String("api", "http://localhost:8080", "backend API URL to proxy to")
	flag.Parse()

	// Parse the API URL for the reverse proxy
	apiTarget, err := url.Parse(*apiURL)
	if err != nil {
		log.Fatalf("invalid API URL %q: %v", *apiURL, err)
	}

	// Load embedded frontend
	frontendFS, err := web.DistFS()
	if err != nil {
		log.Fatalf("failed to load embedded frontend: %v", err)
	}

	// Create reverse proxy for API requests
	apiProxy := httputil.NewSingleHostReverseProxy(apiTarget)

	// Create the main handler
	mux := http.NewServeMux()

	// Proxy /api requests to the backend
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		apiProxy.ServeHTTP(w, r)
	})

	// Health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("ok"))
	})

	// Serve frontend for all other requests
	fileServer := http.FileServer(http.FS(frontendFS))
	mux.HandleFunc("/", spaHandler(frontendFS, fileServer))

	// Wrap the entire handler with security headers middleware
	handler := shared.SecurityHeadersMiddleware(mux)

	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Longer for proxied requests
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("frontend: serving on %s", *addr)
		log.Printf("frontend: proxying /api to %s", *apiURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("frontend: listen error: %v", err)
		}
	}()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("frontend: shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("frontend: shutdown error: %v", err)
	}
	log.Println("frontend: stopped")
}

// setCacheHeaders sets appropriate Cache-Control headers based on file type
func setCacheHeaders(w http.ResponseWriter, path string) {
	// Service worker file - no caching at all to ensure immediate updates
	if path == "sw.js" {
		// no-store prevents any caching, ensuring the service worker is always fetched fresh
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		return
	}

	// HTML files (index.html) - use no-cache to allow revalidation
	// This ensures fresh CSP headers while still allowing browser caching for performance
	if strings.HasSuffix(path, ".html") {
		// no-cache means "revalidate before using" - allows caching but ensures fresh content
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		return
	}

	// Versioned assets in _app/immutable/ - cache aggressively (1 year)
	// These files have content hashes in their filenames, so they're safe to cache long-term
	if strings.Contains(path, "_app/immutable/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}

	// Static assets (SVG, PNG, etc.) - cache with revalidation (1 week)
	// These don't change often but should be revalidated periodically
	if strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".jpeg") ||
		strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".webp") ||
		strings.HasSuffix(path, ".ico") {
		w.Header().Set("Cache-Control", "public, max-age=604800, must-revalidate")
		return
	}

	// Other assets (JS, CSS not in immutable) - cache with revalidation (1 day)
	// Fallback for any other static assets
	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
		w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
		return
	}
}

// spaHandler serves static files with SPA fallback to index.html
func spaHandler(fsys fs.FS, fileServer http.Handler) http.HandlerFunc {
	// Pre-read index.html for SPA fallback
	indexHTML, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		log.Printf("warning: could not read index.html: %v", err)
		indexHTML = []byte(
			"<!DOCTYPE html><html><body>Frontend not found. Run: make build-frontend</body></html>",
		)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Service worker must be served with correct MIME type
		if path == "sw.js" {
			// Try to open the file
			f, err := fsys.Open(path)
			if err == nil {
				_ = f.Close() // Ignore error - just checking if file exists
				// Set cache headers and MIME type for service worker
				setCacheHeaders(w, path)
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
				fileServer.ServeHTTP(w, r)
				return
			}
			// Service worker not found
			http.NotFound(w, r)
			return
		}

		// Try to open the file
		f, err := fsys.Open(path)
		if err == nil {
			_ = f.Close() // Ignore error - just checking if file exists
			// Set cache headers before serving the file
			setCacheHeaders(w, path)
			fileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist - check if it looks like an asset (has extension)
		if strings.Contains(path, ".") {
			http.NotFound(w, r)
			return
		}

		// Serve index.html for SPA client-side routing
		setCacheHeaders(w, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write(indexHTML)
	}
}
