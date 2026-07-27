package netbird

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func daemonTestServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	file, err := os.CreateTemp("/tmp", "nb-json-")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close(); _ = os.Remove(path) })
	go func() { _ = http.Serve(listener, handler) }()
	return path
}

func TestDaemonJSONClientUsesPrivateStatusRPC(t *testing.T) {
	socket := daemonTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/daemon.DaemonService/Status" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "getFullPeerStatus") {
			t.Fatalf("missing fixed status request: %s", body)
		}
		_, _ = w.Write([]byte(`{"status":"Connected","fullStatus":{"managementState":{"connected":true},"signalState":{"connected":true},"localPeerState":{"IP":"100.64.0.1","pubKey":"local","fqdn":"nas"},"peers":[{"pubKey":"peer","IP":"100.64.0.2","connStatus":"Connected","fqdn":"peer.example"}]}}`))
	}))
	client, err := NewDaemonJSONClient("unix://"+socket, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	status := client.Status(context.Background())
	if !status.Connected || status.State != "Connected" {
		t.Fatalf("unexpected status: %#v", status)
	}
	raw, err := client.StatusJSON(context.Background())
	if err != nil || !strings.Contains(string(raw), `"netbirdIp":"100.64.0.1"`) {
		t.Fatalf("unexpected normalized status: %s, %v", raw, err)
	}
}

func TestDaemonJSONClientRejectsNonUnixAddress(t *testing.T) {
	if _, err := NewDaemonJSONClient("tcp://127.0.0.1:8080", time.Second); err == nil {
		t.Fatal("TCP daemon gateway must not be accepted")
	}
}
