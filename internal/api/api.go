package api

import (
	"context"
	"encoding/json"
	"io"
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
type clientManager interface {
	statusProvider
	Connect(context.Context, netbird.ConnectOptions) error
	Disconnect(context.Context) error
	Profiles(context.Context) ([]netbird.Profile, error)
	AddProfile(context.Context, string) error
	SelectProfile(context.Context, string) error
	RenameProfile(context.Context, string, string) error
	RemoveProfile(context.Context, string) error
	Networks(context.Context) ([]netbird.Network, error)
	SelectNetworks(context.Context, []string, bool) error
	DeselectNetworks(context.Context, []string) error
	Diagnose(context.Context) (map[string]any, error)
}
type binaryManager interface {
	Resolve(context.Context) netbird.Binary
	Versions(context.Context) ([]netbird.Binary, error)
	Switch(context.Context, string) (netbird.Binary, error)
	Rollback(context.Context) (netbird.Binary, error)
	UseBundled() error
	Delete(context.Context, string) error
}
type lifecycle interface {
	Check(context.Context, string, string) (netbird.Release, error)
	Download(context.Context, string, string, string) (netbird.Binary, error)
	Upload(context.Context, string, io.Reader, bool) (netbird.Binary, error)
	Sources() ([]netbird.DownloadSource, error)
	SaveSource(netbird.DownloadSource) (netbird.DownloadSource, error)
	DeleteSource(string) error
	TestSource(context.Context, string) error
}
type response struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

func NewHandler(logger *slog.Logger, client statusProvider, args ...any) http.Handler {
	features, _ := client.(clientManager)
	var manager binaryManager
	var life lifecycle
	var build BuildInfo
	if len(args) == 1 {
		build, _ = args[0].(BuildInfo)
	} else if len(args) == 2 {
		manager, _ = args[0].(binaryManager)
		build, _ = args[1].(BuildInfo)
	} else if len(args) == 3 {
		manager, _ = args[0].(binaryManager)
		life, _ = args[1].(lifecycle)
		build, _ = args[2].(BuildInfo)
	}
	if manager == nil {
		manager = unavailableManager{}
	}
	if life == nil {
		life = unavailableLifecycle{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, response{Status: "ok", Data: map[string]string{"service": "netbird-fnos-api"}})
	})
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, response{Status: "ok", Data: build}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, response{Status: "ok", Data: client.Status(r.Context())})
	})
	mux.HandleFunc("POST /api/connect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if features == nil {
			writeError(w, 503, "client unavailable")
			return
		}
		var o netbird.ConnectOptions
		if !decode(w, r, &o) {
			return
		}
		if e := features.Connect(r.Context(), o); e != nil {
			writeError(w, 409, "connect failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: features.Status(r.Context())})
	})
	mux.HandleFunc("POST /api/disconnect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if features == nil {
			writeError(w, 503, "client unavailable")
			return
		}
		if e := features.Disconnect(r.Context()); e != nil {
			writeError(w, 409, "disconnect failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: features.Status(r.Context())})
	})
	mux.HandleFunc("GET /api/profiles", func(w http.ResponseWriter, r *http.Request) {
		if features == nil {
			writeError(w, 503, "client unavailable")
			return
		}
		v, e := features.Profiles(r.Context())
		if e != nil {
			writeError(w, 409, "profiles unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("POST /api/profiles", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := features.AddProfile(r.Context(), b.Name); e != nil {
			writeError(w, 400, "profile add failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/profiles/{handle}/select", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		if e := features.SelectProfile(r.Context(), r.PathValue("handle")); e != nil {
			writeError(w, 400, "profile selection failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("PUT /api/profiles/{handle}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := features.RenameProfile(r.Context(), r.PathValue("handle"), b.Name); e != nil {
			writeError(w, 400, "profile rename failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("DELETE /api/profiles/{handle}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		if e := features.RemoveProfile(r.Context(), r.PathValue("handle")); e != nil {
			writeError(w, 400, "profile removal failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("GET /api/networks", func(w http.ResponseWriter, r *http.Request) {
		if features == nil {
			writeError(w, 503, "client unavailable")
			return
		}
		v, e := features.Networks(r.Context())
		if e != nil {
			writeError(w, 409, "networks unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("POST /api/networks/select", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b struct {
			IDs    []string `json:"ids"`
			Append bool     `json:"append"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := features.SelectNetworks(r.Context(), b.IDs, b.Append); e != nil {
			writeError(w, 400, "network selection failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/networks/deselect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b struct {
			IDs []string `json:"ids"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := features.DeselectNetworks(r.Context(), b.IDs); e != nil {
			writeError(w, 400, "network deselection failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("GET /api/diagnostics", func(w http.ResponseWriter, r *http.Request) {
		if features == nil {
			writeError(w, 503, "client unavailable")
			return
		}
		v, _ := features.Diagnose(r.Context())
		writeJSON(w, response{Status: "ok", Data: v})
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
	mux.HandleFunc("POST /api/client/check-update", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var b struct {
			Version string `json:"version"`
			Arch    string `json:"arch"`
		}
		if !decode(w, r, &b) {
			return
		}
		value, e := life.Check(r.Context(), b.Version, b.Arch)
		if e != nil {
			writeError(w, 400, "official release unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: value})
	})
	mux.HandleFunc("POST /api/client/download", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var b struct {
			Version  string `json:"version"`
			Arch     string `json:"arch"`
			SourceID string `json:"sourceId"`
		}
		if !decode(w, r, &b) {
			return
		}
		value, e := life.Download(r.Context(), b.Version, b.Arch, b.SourceID)
		if e != nil {
			writeError(w, 400, "download or checksum validation failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: value})
	})
	mux.HandleFunc("POST /api/client/upload", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if e := r.ParseMultipartForm(64 << 20); e != nil {
			writeError(w, 400, "invalid upload")
			return
		}
		f, h, e := r.FormFile("file")
		if e != nil {
			writeError(w, 400, "file required")
			return
		}
		defer f.Close()
		value, e := life.Upload(r.Context(), h.Filename, f, r.FormValue("allowUnverified") == "true")
		if e != nil {
			writeError(w, 400, "upload rejected")
			return
		}
		writeJSON(w, response{Status: "ok", Data: value})
	})
	mux.HandleFunc("GET /api/download-sources", func(w http.ResponseWriter, r *http.Request) {
		all, e := life.Sources()
		if e != nil {
			writeError(w, 500, "cannot read sources")
			return
		}
		writeJSON(w, response{Status: "ok", Data: all})
	})
	mux.HandleFunc("POST /api/download-sources", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var s netbird.DownloadSource
		if !decode(w, r, &s) {
			return
		}
		v, e := life.SaveSource(s)
		if e != nil {
			writeError(w, 400, "invalid source")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("PUT /api/download-sources/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var s netbird.DownloadSource
		if !decode(w, r, &s) {
			return
		}
		s.ID = r.PathValue("id")
		v, e := life.SaveSource(s)
		if e != nil {
			writeError(w, 400, "invalid source")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("DELETE /api/download-sources/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if e := life.DeleteSource(r.PathValue("id")); e != nil {
			writeError(w, 404, "source not found")
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]string{"deleted": r.PathValue("id")}})
	})
	mux.HandleFunc("POST /api/download-sources/{id}/test", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if e := life.TestSource(r.Context(), r.PathValue("id")); e != nil {
			writeError(w, 400, "source test failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]bool{"reachable": true}})
	})
	return logging(logger, mux)
}

type unavailableManager struct{}
type unavailableLifecycle struct{}

func (unavailableLifecycle) Check(context.Context, string, string) (netbird.Release, error) {
	return netbird.Release{}, os.ErrNotExist
}
func (unavailableLifecycle) Download(context.Context, string, string, string) (netbird.Binary, error) {
	return netbird.Binary{}, os.ErrNotExist
}
func (unavailableLifecycle) Upload(context.Context, string, io.Reader, bool) (netbird.Binary, error) {
	return netbird.Binary{}, os.ErrNotExist
}
func (unavailableLifecycle) Sources() ([]netbird.DownloadSource, error) {
	return []netbird.DownloadSource{}, nil
}
func (unavailableLifecycle) SaveSource(netbird.DownloadSource) (netbird.DownloadSource, error) {
	return netbird.DownloadSource{}, os.ErrNotExist
}
func (unavailableLifecycle) DeleteSource(string) error                { return os.ErrNotExist }
func (unavailableLifecycle) TestSource(context.Context, string) error { return os.ErrNotExist }

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
