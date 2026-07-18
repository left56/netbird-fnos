package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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
		path := strings.TrimPrefix(r.URL.Path, prefix)
		if path == "" {
			path = "/"
		}
		if strings.HasPrefix(path, "/api/") {
			http.StripPrefix(prefix, api).ServeHTTP(w, r)
			return
		}
		request := r.Clone(r.Context())
		request.URL.Path = path
		files.ServeHTTP(w, request)
	})
}
