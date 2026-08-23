package api

import (
	"food-lab-chain-service/health"
	"food-lab-chain-service/store"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func NewRouter(s *store.Store, webFS fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Handler)
	mux.HandleFunc("/api/specimens", listSpecimens(s))
	mux.HandleFunc("/api/specimens/", handoffSpecimen(s))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		asset := r.URL.Path
		if asset == "/" {
			asset = "/index.html"
		}
		data, err := fs.ReadFile(webFS, path.Clean(strings.TrimPrefix(asset, "/")))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(asset, ".js") {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = w.Write(data)
	})
	return mux
}
func specimenID(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, "/api/specimens/"), "/handoff")
}
