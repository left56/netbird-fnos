package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/left56/netbird-fnos/internal/netbird"
	"github.com/left56/netbird-fnos/internal/netbird/parser"
)

type fakeStatus struct{}

func (fakeStatus) Status(context.Context) netbird.Status {
	return netbird.Status{State: "unavailable", Detail: "official NetBird CLI is not installed"}
}

func TestStatusHasStableUnavailableResponse(t *testing.T) {
	h := NewHandler(testLogger(), fakeStatus{}, BuildInfo{Version: "test"})
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("got %d", r.Code)
	}
	var payload struct {
		Data netbird.RuntimeStatus `json:"data"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Connection.Connected || payload.Data.Connection.Management != "unavailable" {
		t.Fatalf("unexpected stable unavailable response: %s", r.Body.String())
	}
}

type fakeRuntimeService struct{}

func (fakeRuntimeService) Get(context.Context) (netbird.RuntimeStatus, error) {
	return netbird.RuntimeStatus{Connection: netbird.RuntimeConnection{Connected: true}, Statistics: netbird.RuntimeStatistics{PeerCount: 1}}, nil
}

type fakePeerService struct{}

func (fakePeerService) List(context.Context) ([]parser.Peer, error) {
	return []parser.Peer{{ID: "peer-1", Connected: true}}, nil
}

type fakeNetworkService struct{}

func (fakeNetworkService) List(context.Context) (netbird.NetworkList, error) {
	return netbird.NetworkList{All: []netbird.Network{{ID: "n1", Name: "office"}}, Capabilities: netbird.Capabilities{Networks: true}}, nil
}
func (fakeNetworkService) Select(context.Context, []string, bool) error { return nil }
func (fakeNetworkService) Deselect(context.Context, []string) error     { return nil }

func TestRuntimeEndpointsUseServices(t *testing.T) {
	h := NewHandler(testLogger(), fakeStatus{}, nil, nil, nil, fakeRuntimeService{}, fakePeerService{}, fakeNetworkService{}, BuildInfo{})
	for _, route := range []string{"/api/status", "/api/peers", "/api/networks"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, route, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", route, r.Code, r.Body.String())
		}
	}
}

func TestLogReaderRedactsSensitiveLines(t *testing.T) {
	file := filepath.Join(t.TempDir(), "api.log")
	mustWrite(t, file, "safe line\nsetup_key=do-not-return\n")
	lines, err := NewLogReader(file).Latest()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(lines, "\n"), "do-not-return") {
		t.Fatalf("secret leaked in %q", lines)
	}
}

func TestLogReaderIncludesDaemonLog(t *testing.T) {
	dir := t.TempDir()
	apiLog, daemonLog := filepath.Join(dir, "api.log"), filepath.Join(dir, "daemon.log")
	mustWrite(t, apiLog, "api started\n")
	mustWrite(t, daemonLog, "daemon started\n")
	lines, err := NewLogReader(apiLog, daemonLog).Latest()
	if err != nil || !strings.Contains(strings.Join(lines, "\n"), "daemon started") {
		t.Fatalf("unexpected daemon log result: %#v, %v", lines, err)
	}
}

func TestStaticFilesThroughGateway(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "index.html"), "<!doctype html><div id=app></div>")
	mustWrite(t, filepath.Join(root, "assets", "test.js"), "console.log('netbird')")
	mustWrite(t, filepath.Join(root, "assets", "test.css"), "body { color: black; }")
	h := WithStaticFiles(NewHandler(testLogger(), fakeStatus{}, BuildInfo{Version: "test"}), "/app/netbird-fnos", root)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		contentType string
		body        string
	}{
		{"gateway entry", "/app/netbird-fnos", http.StatusOK, "text/html", "<!doctype html>"},
		{"gateway entry slash", "/app/netbird-fnos/", http.StatusOK, "text/html", "<!doctype html>"},
		{"javascript asset", "/app/netbird-fnos/assets/test.js", http.StatusOK, "javascript", "console.log"},
		{"stylesheet asset", "/app/netbird-fnos/assets/test.css", http.StatusOK, "text/css", "color: black"},
		{"missing asset is not SPA fallback", "/app/netbird-fnos/assets/missing.js", http.StatusNotFound, "", "404 page not found"},
		{"SPA route fallback", "/app/netbird-fnos/settings", http.StatusOK, "text/html", "<!doctype html>"},
		{"gateway API", "/app/netbird-fnos/api/health", http.StatusOK, "application/json", "netbird-fnos-api"},
		{"direct API remains API", "/api/health", http.StatusOK, "application/json", "netbird-fnos-api"},
		{"outside gateway is not static", "/app/assets/test.js", http.StatusNotFound, "", "404 page not found"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, test.path, nil))
			if r.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", r.Code, test.wantStatus)
			}
			if test.contentType != "" && !strings.Contains(r.Header().Get("Content-Type"), test.contentType) {
				t.Fatalf("Content-Type = %q, want it to contain %q", r.Header().Get("Content-Type"), test.contentType)
			}
			if !strings.Contains(r.Body.String(), test.body) {
				t.Fatalf("body = %q, want it to contain %q", r.Body.String(), test.body)
			}
		})
	}
}

func TestStaticFilesWithBuiltFrontend(t *testing.T) {
	if os.Getenv("NETBIRD_FNOS_VERIFY_BUILT_FRONTEND") != "1" {
		t.Skip("built frontend verification runs through make verify-static-files")
	}
	root := filepath.Join("..", "..", "app", "www")
	for _, pattern := range []string{"*.js", "*.css"} {
		assets, err := filepath.Glob(filepath.Join(root, "assets", pattern))
		if err != nil || len(assets) == 0 {
			t.Fatalf("built frontend has no %s asset: %v", pattern, err)
		}
		asset := filepath.Base(assets[0])
		h := WithStaticFiles(NewHandler(testLogger(), fakeStatus{}, BuildInfo{Version: "test"}), "/app/netbird-fnos", root)
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/app/netbird-fnos/assets/"+asset, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("built asset %q returned %d", asset, r.Code)
		}
		if pattern == "*.js" && !strings.Contains(r.Header().Get("Content-Type"), "javascript") {
			t.Fatalf("JavaScript Content-Type = %q", r.Header().Get("Content-Type"))
		}
		if pattern == "*.css" && !strings.Contains(r.Header().Get("Content-Type"), "text/css") {
			t.Fatalf("CSS Content-Type = %q", r.Header().Get("Content-Type"))
		}
	}
}

func mustWrite(t *testing.T, filename, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
