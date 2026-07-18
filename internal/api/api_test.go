package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/left56/netbird-fnos/internal/netbird"
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
	want := `{"status":"ok","data":{"state":"unavailable","connected":false,"detail":"official NetBird CLI is not installed"}}` + "\n"
	if r.Body.String() != want {
		t.Fatalf("got %s", r.Body.String())
	}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
