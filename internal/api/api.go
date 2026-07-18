package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
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
type binaryManager interface {
	Resolve(context.Context) netbird.Binary
	Versions(context.Context) ([]netbird.Binary, error)
	Switch(context.Context, string) (netbird.Binary, error)
	Rollback(context.Context) (netbird.Binary, error)
	UseBundled() error
	Delete(context.Context, string) error
}
type response struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

func NewHandler(logger *slog.Logger, client statusProvider, args ...any) http.Handler {
	var manager binaryManager
	var build BuildInfo
	if len(args) == 1 {
		build, _ = args[0].(BuildInfo)
	} else if len(args) == 2 {
		manager, _ = args[0].(binaryManager)
		build, _ = args[1].(BuildInfo)
	}
	if manager == nil {
		manager = unavailableManager{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, response{Status: "ok", Data: map[string]string{"service": "netbird-fnos-api"}})
	})
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, response{Status: "ok", Data: build}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, response{Status: "ok", Data: client.Status(r.Context())})
	})
	mux.HandleFunc("GET /api/client", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, response{Status: "ok", Data: manager.Resolve(r.Context())})
	})
	mux.HandleFunc("GET /api/client/versions", func(w http.ResponseWriter, r *http.Request) {
		versions, err := manager.Versions(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot list versions")
			return
		}
		writeJSON(w, response{Status: "ok", Data: versions})
	})
	mux.HandleFunc("POST /api/client/switch", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var body struct {
			Version string `json:"version"`
		}
		if !decode(w, r, &body) {
			return
		}
		b, err := manager.Switch(r.Context(), body.Version)
		if err != nil {
			writeError(w, http.StatusBadRequest, "switch rejected")
			return
		}
		writeJSON(w, response{Status: "ok", Data: b})
	})
	mux.HandleFunc("POST /api/client/rollback", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		b, err := manager.Rollback(r.Context())
		if err != nil {
			writeError(w, http.StatusConflict, "rollback unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: b})
	})
	mux.HandleFunc("POST /api/client/use-bundled", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if err := manager.UseBundled(); err != nil {
			writeError(w, http.StatusInternalServerError, "unable to select bundled client")
			return
		}
		writeJSON(w, response{Status: "ok", Data: manager.Resolve(r.Context())})
	})
	mux.HandleFunc("DELETE /api/client/versions/{version}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if err := manager.Delete(r.Context(), r.PathValue("version")); err != nil {
			writeError(w, http.StatusConflict, "version cannot be deleted")
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]string{"deleted": r.PathValue("version")}})
	})
	// These lifecycle endpoints intentionally exist now so the UI never reaches
	// an arbitrary URL/executable API. Download and upload staging are enabled
	// only after an official checksum is supplied by the release resolver.
	for _, p := range []string{"/api/client/check-update", "/api/client/download", "/api/client/upload"} {
		mux.HandleFunc("POST "+p, func(w http.ResponseWriter, r *http.Request) {
			if !admin(w, r) {
				return
			}
			writeError(w, http.StatusNotImplemented, "official release staging is not configured")
		})
	}
	mux.HandleFunc("GET /api/download-sources", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, response{Status: "ok", Data: []any{}}) })
	for _, p := range []string{"POST /api/download-sources", "PUT /api/download-sources/{id}", "DELETE /api/download-sources/{id}", "POST /api/download-sources/{id}/test"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			if !admin(w, r) {
				return
			}
			writeError(w, http.StatusNotImplemented, "download source configuration is not configured")
		})
	}
	return logging(logger, mux)
}

type unavailableManager struct{}

func (unavailableManager) Resolve(context.Context) netbird.Binary {
	return netbird.Binary{Source: netbird.Missing}
}
func (unavailableManager) Versions(context.Context) ([]netbird.Binary, error) {
	return []netbird.Binary{}, nil
}
func (unavailableManager) Switch(context.Context, string) (netbird.Binary, error) {
	return netbird.Binary{}, os.ErrNotExist
}
func (unavailableManager) Rollback(context.Context) (netbird.Binary, error) {
	return netbird.Binary{}, os.ErrNotExist
}
func (unavailableManager) UseBundled() error                    { return os.ErrNotExist }
func (unavailableManager) Delete(context.Context, string) error { return os.ErrNotExist }

func writeJSON(w http.ResponseWriter, value response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response{Status: "error", Data: map[string]string{"message": message}})
}
func admin(w http.ResponseWriter, r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("X-Trim-Isadmin"), "true") || r.Header.Get("X-Trim-Isadmin") == "1" {
		return true
	}
	writeError(w, http.StatusForbidden, "administrator permission required")
	return false
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body required")
		return false
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return false
	}
	return true
}

var _ = os.ErrNotExist

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
