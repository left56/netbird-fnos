package netbird

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type versionRunner struct{ err error }

func (r versionRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	if len(args) == 1 && args[0] == "version" {
		return []byte("netbird version 0.71.4\n"), nil
	}
	return []byte(`{"status":"Connected","connected":true}`), nil
}
func elfFile(t *testing.T, path string) {
	t.Helper()
	header := make([]byte, 20)
	copy(header, "\x7fELF")
	header[5] = 1
	header[18] = 62
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, header, 0755); err != nil {
		t.Fatal(err)
	}
}
func TestResolveManagedThenBundledFallback(t *testing.T) {
	root := t.TempDir()
	app := t.TempDir()
	bundled := filepath.Join(app, "bin", "netbird")
	elfFile(t, bundled)
	m := NewBinaryManager(root, app, versionRunner{}, time.Second)
	if got := m.Resolve(context.Background()).Source; got != Bundled {
		t.Fatalf("got %s", got)
	}
	managed := filepath.Join(root, "netbird", "versions", "0.71.4", "netbird")
	elfFile(t, managed)
	if err := m.atomicLink("current", managed); err != nil {
		t.Fatal(err)
	}
	if got := m.Resolve(context.Background()).Source; got != Managed {
		t.Fatalf("got %s", got)
	}
	if err := os.Remove(managed); err != nil {
		t.Fatal(err)
	}
	if got := m.Resolve(context.Background()).Source; got != Bundled {
		t.Fatalf("got %s", got)
	}
}
func TestSwitchAndCannotDeleteCurrent(t *testing.T) {
	root := t.TempDir()
	app := t.TempDir()
	elfFile(t, filepath.Join(app, "bin", "netbird"))
	m := NewBinaryManager(root, app, versionRunner{}, time.Second)
	elfFile(t, filepath.Join(root, "netbird", "versions", "0.71.4", "netbird"))
	if _, err := m.Switch(context.Background(), "0.71.4"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(context.Background(), "0.71.4"); err == nil {
		t.Fatal("deleted current")
	}
	if _, err := m.Switch(context.Background(), "../bad"); err == nil {
		t.Fatal("accepted path traversal")
	}
}
func TestInstallUnverifiedRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	app := t.TempDir()
	staged := filepath.Join(t.TempDir(), "netbird")
	elfFile(t, staged)
	m := NewBinaryManager(root, app, versionRunner{}, time.Second)
	_, err := m.Install(context.Background(), staged, "upload-unverified", "", false)
	if err == nil {
		t.Fatal("unverified upload accepted")
	}
}
