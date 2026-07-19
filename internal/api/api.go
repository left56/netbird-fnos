package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/left56/netbird-fnos/internal/netbird"
	"github.com/left56/netbird-fnos/internal/netbird/parser"
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
type profileManager interface {
	List(context.Context) ([]netbird.ProfileDetail, error)
	Get(context.Context, string) (netbird.ProfileDetail, error)
	Create(context.Context, netbird.ProfileCreate) (netbird.ProfileDetail, error)
	Update(context.Context, string, netbird.ProfileUpdate) (map[string]any, error)
	Select(context.Context, string) error
	Connect(context.Context, string) error
	Disconnect(context.Context, string) error
	Rename(context.Context, string, string) error
	Delete(context.Context, string) error
	SetSecret(string, string, string) error
	ClearSecret(string, string) error
}
type runtimeStatusService interface {
	Get(context.Context) (netbird.RuntimeStatus, error)
}
type peerService interface {
	List(context.Context) ([]parser.Peer, error)
}
type networkService interface {
	List(context.Context) (netbird.NetworkList, error)
	Select(context.Context, []string, bool) error
	Deselect(context.Context, []string) error
}
type logReader interface{ Latest() ([]string, error) }
type response struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

func NewHandler(logger *slog.Logger, client statusProvider, args ...any) http.Handler {
	features, _ := client.(clientManager)
	var manager binaryManager
	var life lifecycle
	var profiles profileManager
	var runtime runtimeStatusService
	var peers peerService
	var networks networkService
	var logs logReader
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
	} else if len(args) == 4 {
		manager, _ = args[0].(binaryManager)
		life, _ = args[1].(lifecycle)
		profiles, _ = args[2].(profileManager)
		build, _ = args[3].(BuildInfo)
	} else if len(args) == 7 {
		manager, _ = args[0].(binaryManager)
		life, _ = args[1].(lifecycle)
		profiles, _ = args[2].(profileManager)
		runtime, _ = args[3].(runtimeStatusService)
		peers, _ = args[4].(peerService)
		networks, _ = args[5].(networkService)
		build, _ = args[6].(BuildInfo)
	} else if len(args) == 8 {
		manager, _ = args[0].(binaryManager)
		life, _ = args[1].(lifecycle)
		profiles, _ = args[2].(profileManager)
		runtime, _ = args[3].(runtimeStatusService)
		peers, _ = args[4].(peerService)
		networks, _ = args[5].(networkService)
		logs, _ = args[6].(logReader)
		build, _ = args[7].(BuildInfo)
	}
	if manager == nil {
		manager = unavailableManager{}
	}
	if life == nil {
		life = unavailableLifecycle{}
	}
	if profiles == nil {
		profiles = unavailableProfiles{}
	}
	if runtime == nil {
		runtime = legacyStatusService{client: client}
	}
	if peers == nil {
		peers = unavailablePeers{}
	}
	if networks == nil {
		networks = legacyNetworks{client: features}
	}
	if logs == nil {
		logs = unavailableLogs{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, response{Status: "ok", Data: map[string]string{"service": "netbird-fnos-api"}})
	})
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, response{Status: "ok", Data: build}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		v, err := runtime.Get(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "status unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
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
		v, e := profiles.List(r.Context())
		if e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("GET /api/profiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := profiles.Get(r.Context(), r.PathValue("id"))
		if e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]any{"profile": v.Metadata, "config": v.Config, "secrets": map[string]bool{"setupKeyConfigured": v.Config.SetupKeyConfigured, "presharedKeyConfigured": v.Config.PresharedKeyConfigured}, "runtime": v.Runtime, "source": v.Source, "capabilities": v.Capabilities, "restartRequiredFields": v.RestartRequiredFields}})
	})
	mux.HandleFunc("POST /api/profiles", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b netbird.ProfileCreate
		if !decode(w, r, &b) {
			return
		}
		v, e := profiles.Create(r.Context(), b)
		if e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]any{"created": true, "selected": v.Runtime.Active, "connected": v.Runtime.Connected, "warnings": []string{}}})
	})
	mux.HandleFunc("POST /api/profiles/{handle}/select", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		if e := profiles.Select(r.Context(), r.PathValue("handle")); e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("PUT /api/profiles/{handle}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		var b netbird.ProfileUpdate
		if !decode(w, r, &b) {
			return
		}
		v, e := profiles.Update(r.Context(), r.PathValue("handle"), b)
		if e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("DELETE /api/profiles/{handle}", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) || features == nil {
			return
		}
		if e := profiles.Delete(r.Context(), r.PathValue("handle")); e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/profiles/{id}/connect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if e := profiles.Connect(r.Context(), r.PathValue("id")); e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/profiles/{id}/disconnect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		if e := profiles.Disconnect(r.Context(), r.PathValue("id")); e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/profiles/{id}/rename", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var b struct {
			Name string `json:"name"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := profiles.Rename(r.Context(), r.PathValue("id"), b.Name); e != nil {
			profileFailure(w, e)
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	for _, spec := range []struct{ kind string }{{"setup-key"}, {"preshared-key"}} {
		kind := spec.kind
		mux.HandleFunc("PUT /api/profiles/{id}/secrets/"+kind, func(w http.ResponseWriter, r *http.Request) {
			if !admin(w, r) {
				return
			}
			var b struct {
				Value string `json:"value"`
			}
			if !decode(w, r, &b) {
				return
			}
			if b.Value == "" {
				writeProfileError(w, 422, "PROFILE_SECRET_INVALID", "密钥不能为空")
				return
			}
			if e := profiles.SetSecret(r.PathValue("id"), kind, b.Value); e != nil {
				writeProfileError(w, 422, "PROFILE_SECRET_INVALID", "密钥无效")
				return
			}
			writeJSON(w, response{Status: "ok", Data: true})
		})
		mux.HandleFunc("DELETE /api/profiles/{id}/secrets/"+kind, func(w http.ResponseWriter, r *http.Request) {
			if !admin(w, r) {
				return
			}
			if e := profiles.ClearSecret(r.PathValue("id"), kind); e != nil {
				profileFailure(w, e)
				return
			}
			writeJSON(w, response{Status: "ok", Data: true})
		})
	}
	mux.HandleFunc("GET /api/networks", func(w http.ResponseWriter, r *http.Request) {
		v, e := networks.List(r.Context())
		if e != nil {
			writeError(w, 409, "networks unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("POST /api/networks/select", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var b struct {
			IDs    []string `json:"ids"`
			Append bool     `json:"append"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := networks.Select(r.Context(), b.IDs, b.Append); e != nil {
			writeError(w, 400, "network selection failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("POST /api/networks/deselect", func(w http.ResponseWriter, r *http.Request) {
		if !admin(w, r) {
			return
		}
		var b struct {
			IDs []string `json:"ids"`
		}
		if !decode(w, r, &b) {
			return
		}
		if e := networks.Deselect(r.Context(), b.IDs); e != nil {
			writeError(w, 400, "network deselection failed")
			return
		}
		writeJSON(w, response{Status: "ok", Data: true})
	})
	mux.HandleFunc("GET /api/peers", func(w http.ResponseWriter, r *http.Request) {
		v, e := peers.List(r.Context())
		if e != nil {
			writeError(w, http.StatusServiceUnavailable, "peers unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: v})
	})
	mux.HandleFunc("GET /api/logs/latest", func(w http.ResponseWriter, _ *http.Request) {
		lines, err := logs.Latest()
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "logs unavailable")
			return
		}
		writeJSON(w, response{Status: "ok", Data: map[string]any{"lines": lines}})
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
type unavailableProfiles struct{}
type unavailablePeers struct{}
type unavailableLogs struct{}

// These adapters keep the small handler test constructor backward compatible.
// The production entrypoint always injects the dedicated runtime services.
type legacyStatusService struct{ client statusProvider }

func (s legacyStatusService) Get(ctx context.Context) (netbird.RuntimeStatus, error) {
	v := s.client.Status(ctx)
	return netbird.RuntimeStatus{Connection: netbird.RuntimeConnection{Connected: v.Connected, Management: v.State}}, nil
}

type legacyNetworks struct{ client clientManager }

func (s legacyNetworks) List(ctx context.Context) (netbird.NetworkList, error) {
	if s.client == nil {
		return netbird.NetworkList{}, os.ErrNotExist
	}
	v, e := s.client.Networks(ctx)
	return netbird.NetworkList{All: v, Overlapping: []netbird.Network{}, ExitNodes: []netbird.Network{}, Selected: []netbird.Network{}, Pending: []netbird.Network{}}, e
}
func (s legacyNetworks) Select(ctx context.Context, ids []string, appendMode bool) error {
	if s.client == nil {
		return os.ErrNotExist
	}
	return s.client.SelectNetworks(ctx, ids, appendMode)
}
func (s legacyNetworks) Deselect(ctx context.Context, ids []string) error {
	if s.client == nil {
		return os.ErrNotExist
	}
	return s.client.DeselectNetworks(ctx, ids)
}
func (unavailablePeers) List(context.Context) ([]parser.Peer, error) { return nil, os.ErrNotExist }
func (unavailableLogs) Latest() ([]string, error)                    { return []string{}, nil }

// NewLogReader provides a bounded, redacted view of the wrapper log. The
// netbird CLI is never asked for logs and secrets are removed defensively.
func NewLogReader(filenames ...string) logReader { return fileLogReader{filenames: filenames} }

type fileLogReader struct{ filenames []string }

func (r fileLogReader) Latest() ([]string, error) {
	all := []string{}
	read := false
	for _, filename := range r.filenames {
		data, err := os.ReadFile(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		read = true
		all = append(all, strings.Split(string(data), "\n")...)
	}
	if !read {
		return nil, os.ErrNotExist
	}
	if len(all) > 100 {
		all = all[len(all)-100:]
	}
	for i := range all {
		all[i] = redactLogLine(all[i])
	}
	return all, nil
}
func redactLogLine(line string) string {
	lower := strings.ToLower(line)
	for _, marker := range []string{"setupkey", "setup_key", "preshared", "token", "privatekey", "private_key"} {
		if strings.Contains(lower, marker) {
			return "[sensitive log entry redacted]"
		}
	}
	return line
}

func (unavailableProfiles) List(context.Context) ([]netbird.ProfileDetail, error) {
	return nil, os.ErrNotExist
}
func (unavailableProfiles) Get(context.Context, string) (netbird.ProfileDetail, error) {
	return netbird.ProfileDetail{}, os.ErrNotExist
}
func (unavailableProfiles) Create(context.Context, netbird.ProfileCreate) (netbird.ProfileDetail, error) {
	return netbird.ProfileDetail{}, os.ErrNotExist
}
func (unavailableProfiles) Update(context.Context, string, netbird.ProfileUpdate) (map[string]any, error) {
	return nil, os.ErrNotExist
}
func (unavailableProfiles) Select(context.Context, string) error         { return os.ErrNotExist }
func (unavailableProfiles) Connect(context.Context, string) error        { return os.ErrNotExist }
func (unavailableProfiles) Disconnect(context.Context, string) error     { return os.ErrNotExist }
func (unavailableProfiles) Rename(context.Context, string, string) error { return os.ErrNotExist }
func (unavailableProfiles) Delete(context.Context, string) error         { return os.ErrNotExist }
func (unavailableProfiles) SetSecret(string, string, string) error       { return os.ErrNotExist }
func (unavailableProfiles) ClearSecret(string, string) error             { return os.ErrNotExist }

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
func writeProfileError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "error", "error": map[string]string{"code": code, "message": message}})
}
func profileFailure(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, os.ErrNotExist):
		writeProfileError(w, 404, "PROFILE_NOT_FOUND", "Profile 不存在")
	case strings.Contains(e.Error(), "already exists"):
		writeProfileError(w, 409, "PROFILE_ALREADY_EXISTS", "Profile 名称已存在")
	case strings.Contains(e.Error(), "cannot be deleted"):
		writeProfileError(w, 409, "PROFILE_ACTIVE", "当前 Profile 不能删除")
	case strings.Contains(e.Error(), "invalid"):
		writeProfileError(w, 422, "PROFILE_CONFIG_INVALID", "Profile 配置无效")
	default:
		writeProfileError(w, 409, "PROFILE_WRITE_FAILED", "Profile 操作失败")
	}
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
