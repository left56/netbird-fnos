package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/left56/netbird-fnos/internal/netbird"
)

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}
type statusProvider interface {
	Status(context.Context) netbird.Status
}
type response struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

func NewHandler(logger *slog.Logger, client statusProvider, build BuildInfo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, response{Status: "ok", Data: map[string]string{"service": "netbird-fnos-api"}})
	})
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, response{Status: "ok", Data: build}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, response{Status: "ok", Data: client.Status(r.Context())})
	})
	return logging(logger, mux)
}

func writeJSON(w http.ResponseWriter, value response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}
func logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("HTTP request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func WithStaticFiles(api http.Handler, prefix, root string) http.Handler {
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := r.URL.Path
		if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") {
			api.ServeHTTP(w, r)
			return
		}
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			relativePath := strings.TrimPrefix(requestPath, prefix)
			if relativePath == "" {
				relativePath = "/"
			}
			if hasParentPathSegment(relativePath) {
				http.NotFound(w, r)
				return
			}
			if relativePath == "/api" || strings.HasPrefix(relativePath, "/api/") {
				http.StripPrefix(prefix, api).ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(relativePath, "/assets/") {
				request := r.Clone(r.Context())
				request.URL.Path = relativePath
				files.ServeHTTP(w, request)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.NotFound(w, r)
				return
			}
			if path.Ext(relativePath) != "" {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		http.NotFound(w, r)
	})
}

func hasParentPathSegment(requestPath string) bool {
	for _, segment := range strings.Split(requestPath, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}
