package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/schemadoctor/web"
)

// RegisterStaticFiles registers embedded web studio assets to be served on the root path.
func RegisterStaticFiles(mux *http.ServeMux) {
	distFS, err := web.GetFS()
	if err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Embedded studio assets unavailable", http.StatusInternalServerError)
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Do not intercept API or health endpoints
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/health") {
			return
		}

		// Check if file exists in FS
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if f, err := distFS.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Single-Page Application (SPA) fallback to index.html for client routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// Dummy interface to avoid unused import
var _ fs.FS
