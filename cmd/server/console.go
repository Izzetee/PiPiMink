package server

import (
	"io/fs"
	"net/http"
	"strings"

	"PiPiMink/web"
)

// setupConsoleRoutes registers the React console UI routes.
func (s *Server) setupConsoleRoutes() {
	// Serve the embedded React console build as an SPA under /console/
	distFS, err := fs.Sub(web.ConsoleFS, "console/dist")
	if err != nil {
		// dist directory not present (dev mode without a build) — skip console routes
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	s.router.PathPrefix("/console").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /console prefix to match files in dist/
		path := strings.TrimPrefix(r.URL.Path, "/console")
		if path == "" {
			http.Redirect(w, r, "/console/", http.StatusMovedPermanently)
			return
		}
		path = strings.TrimPrefix(path, "/")

		// Try to serve the file directly. If it doesn't exist, serve index.html
		// for SPA client-side routing.
		if path != "" {
			if _, err := fs.Stat(distFS, path); err == nil {
				http.StripPrefix("/console", fileServer).ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for all unmatched routes
		index, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			http.Error(w, "Console not available", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
