package netbird

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

type fakeRunner struct {
	binary string
	args   []string
	output []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, binary string, args ...string) ([]byte, error) {
	f.binary, f.args = binary, args
	return f.output, f.err
}

func TestStatusUsesStructuredCLIOutput(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{"status":"Connected","connected":true}`)}
	status := NewClient(runner, "/opt/netbird", time.Second).Status(context.Background())
	if runner.binary != "/opt/netbird" || len(runner.args) != 2 || runner.args[0] != "status" || runner.args[1] != "--json" {
		t.Fatalf("unexpected command: %q %q", runner.binary, runner.args)
	}
	if !status.Connected || status.State != "Connected" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestStatusDoesNotClaimConnectedWhenMissing(t *testing.T) {
	status := NewClient(&fakeRunner{err: exec.ErrNotFound}, "netbird", time.Second).Status(context.Background())
	if status.Connected || status.State != "unavailable" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestStatusDoesNotExposeCommandError(t *testing.T) {
	status := NewClient(&fakeRunner{err: errors.New("sensitive output")}, "netbird", time.Second).Status(context.Background())
	if status.Detail != "official NetBird CLI is unavailable" {
		t.Fatalf("unexpected detail: %q", status.Detail)
	}
}

func TestParseProfilesRecognizesDefaultByName(t *testing.T) {
	profiles := parseProfiles("ID  NAME  ACTIVE\n8fc1e234  default  ✓\n")
	if len(profiles) != 1 || !profiles[0].Default || !profiles[0].Active {
		t.Fatalf("default profile was not recognized: %#v", profiles)
	}
}

func TestRuntimeCommandsUseConfiguredDaemonSocket(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{"status":"Connected","connected":true}`)}
	client := Client{runner: runner, binary: func(context.Context) string { return "/opt/netbird" }, timeout: time.Second, daemonAddr: "unix:///pkg/netbird/daemon.sock"}
	_ = client.Status(context.Background())
	if len(runner.args) != 4 || runner.args[0] != "--daemon-addr" || runner.args[1] != "unix:///pkg/netbird/daemon.sock" || runner.args[2] != "status" {
		t.Fatalf("daemon address was not used: %#v", runner.args)
	}
}

func TestConnectPassesSetupKeyOnlyToOfficialCLI(t *testing.T) {
	runner := &fakeRunner{}
	client := NewClient(runner, "/opt/netbird", time.Second)
	err := client.Connect(context.Background(), ConnectOptions{ManagementURL: "https://netbird.example", SetupKey: "one-time-key"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"up", "--management-url", "https://netbird.example", "--setup-key", "one-time-key"}
	if len(runner.args) != len(want) {
		t.Fatalf("unexpected command: %#v", runner.args)
	}
	for i := range want {
		if runner.args[i] != want[i] {
			t.Fatalf("unexpected command: %#v", runner.args)
		}
	}
}
