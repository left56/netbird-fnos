package netbird

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Source string

const (
	Bundled Source = "bundled"
	Managed Source = "managed"
	Missing Source = "missing"
)

type Capabilities struct {
	Profiles   bool `json:"profiles"`
	Networks   bool `json:"networks"`
	ExitNode   bool `json:"exitNode"`
	JSONStatus bool `json:"jsonStatus"`
}
type Binary struct {
	Source       Source       `json:"source"`
	Path         string       `json:"-"`
	Version      string       `json:"version,omitempty"`
	Arch         string       `json:"arch,omitempty"`
	Checksum     string       `json:"checksum,omitempty"`
	Capabilities Capabilities `json:"capabilities"`
}
type Metadata struct {
	Version      string    `json:"version"`
	Arch         string    `json:"arch"`
	SHA256       string    `json:"sha256"`
	Source       string    `json:"source"`
	DownloadedAt time.Time `json:"downloadedAt"`
	OriginalURL  string    `json:"originalUrl"`
}

// BinaryManager owns all acceptable CLI locations. It intentionally exposes no
// method that takes a browser-provided executable path.
type BinaryManager struct {
	root, bundled string
	runner        Runner
	timeout       time.Duration
	mu            sync.Mutex
}

func NewBinaryManager(pkgvar, appdest string, runner Runner, timeout time.Duration) *BinaryManager {
	return &BinaryManager{root: filepath.Join(pkgvar, "netbird"), bundled: filepath.Join(appdest, "bin", "netbird"), runner: runner, timeout: timeout}
}
func (m *BinaryManager) Path(ctx context.Context) string { return m.Resolve(ctx).Path }
func (m *BinaryManager) Resolve(ctx context.Context) Binary {
	if p := m.linkTarget("current"); p != "" {
		if b, err := m.inspect(ctx, p, Managed); err == nil {
			return b
		}
	}
	if b, err := m.inspect(ctx, m.bundled, Bundled); err == nil {
		return b
	}
	return Binary{Source: Missing, Capabilities: Capabilities{}}
}
func (m *BinaryManager) Versions(ctx context.Context) ([]Binary, error) {
	dirs, err := os.ReadDir(filepath.Join(m.root, "versions"))
	if errors.Is(err, os.ErrNotExist) {
		return []Binary{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Binary, 0, len(dirs))
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		b, err := m.inspect(ctx, filepath.Join(m.root, "versions", d.Name(), "netbird"), Managed)
		if err == nil {
			result = append(result, b)
		}
	}
	return result, nil
}
func (m *BinaryManager) Install(ctx context.Context, staged, source, originalURL string, allowUnverified bool) (Binary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, err := m.inspect(ctx, staged, Managed)
	if err != nil {
		return Binary{}, err
	}
	if source == "upload-unverified" && !allowUnverified {
		return Binary{}, errors.New("unverified upload requires explicit administrator confirmation")
	}
	sum, err := checksum(staged)
	if err != nil {
		return Binary{}, err
	}
	version := safeVersion(b.Version)
	if version == "" {
		return Binary{}, errors.New("unable to determine version")
	}
	dir := filepath.Join(m.root, "versions", version)
	dst := filepath.Join(dir, "netbird")
	if old, err := checksum(dst); err == nil {
		if old != sum {
			return Binary{}, errors.New("version already exists with a different checksum")
		}
		return m.inspect(ctx, dst, Managed)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Binary{}, err
	}
	if err := copyExecutable(staged, dst); err != nil {
		return Binary{}, err
	}
	meta := Metadata{Version: b.Version, Arch: b.Arch, SHA256: sum, Source: source, DownloadedAt: time.Now().UTC(), OriginalURL: originalURL}
	raw, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), raw, 0644); err != nil {
		return Binary{}, err
	}
	b.Checksum = sum
	return b, nil
}
func (m *BinaryManager) Switch(ctx context.Context, version string) (Binary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.switchLocked(ctx, version)
}
func (m *BinaryManager) switchLocked(ctx context.Context, version string) (Binary, error) {
	version = safeVersion(version)
	if version == "" {
		return Binary{}, errors.New("invalid version")
	}
	target := filepath.Join(m.root, "versions", version, "netbird")
	b, err := m.inspect(ctx, target, Managed)
	if err != nil {
		return Binary{}, err
	}
	old := m.linkTarget("current")
	if old != "" {
		if err := m.atomicLink("previous", old); err != nil {
			return Binary{}, err
		}
	}
	if err := m.atomicLink("current", target); err != nil {
		return Binary{}, err
	}
	if _, err := m.inspect(ctx, target, Managed); err != nil {
		_ = m.restorePrevious()
		return Binary{}, err
	}
	return b, nil
}
func (m *BinaryManager) Rollback(ctx context.Context) (Binary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.linkTarget("previous")
	if previous == "" {
		return Binary{}, errors.New("no previous managed version")
	}
	version := filepath.Base(filepath.Dir(previous))
	return m.switchLocked(ctx, version)
}
func (m *BinaryManager) UseBundled() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return os.Remove(filepath.Join(m.root, "current"))
}
func (m *BinaryManager) Delete(ctx context.Context, version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	version = safeVersion(version)
	if version == "" {
		return errors.New("invalid version")
	}
	target := filepath.Join(m.root, "versions", version, "netbird")
	if samePath(target, m.linkTarget("current")) {
		return errors.New("cannot delete current version")
	}
	return os.RemoveAll(filepath.Join(m.root, "versions", version))
}
func (m *BinaryManager) inspect(ctx context.Context, p string, s Source) (Binary, error) {
	info, err := os.Stat(p)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return Binary{}, errors.New("binary is missing or not executable")
	}
	arch, err := elfArch(p)
	if err != nil {
		return Binary{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	out, err := m.runner.Run(ctx, p, "version")
	if err != nil {
		return Binary{}, err
	}
	v := parseVersion(string(out))
	if v == "" {
		return Binary{}, errors.New("invalid netbird version output")
	}
	sum, _ := checksum(p)
	return Binary{Source: s, Path: p, Version: v, Arch: arch, Checksum: sum, Capabilities: Capabilities{Profiles: true, Networks: true, ExitNode: true, JSONStatus: true}}, nil
}
func (m *BinaryManager) linkTarget(name string) string {
	p := filepath.Join(m.root, name)
	dest, err := os.Readlink(p)
	if err != nil {
		return ""
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(m.root, dest)
	}
	clean := filepath.Clean(dest)
	if !strings.HasPrefix(clean, filepath.Join(m.root, "versions")+string(os.PathSeparator)) {
		return ""
	}
	return clean
}
func (m *BinaryManager) atomicLink(name, target string) error {
	if err := os.MkdirAll(m.root, 0755); err != nil {
		return err
	}
	rel, err := filepath.Rel(m.root, target)
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.root, "."+name+".tmp")
	_ = os.Remove(tmp)
	if err = os.Symlink(rel, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(m.root, name))
}
func (m *BinaryManager) restorePrevious() error {
	p := m.linkTarget("previous")
	if p == "" {
		return os.Remove(filepath.Join(m.root, "current"))
	}
	return m.atomicLink("current", p)
}
func checksum(p string) (string, error) {
	f, e := os.Open(p)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func copyExecutable(src, dst string) error {
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if e != nil {
		return e
	}
	_, e = io.Copy(out, in)
	closeErr := out.Close()
	if e != nil {
		return e
	}
	return closeErr
}
func elfArch(p string) (string, error) {
	f, e := os.Open(p)
	if e != nil {
		return "", e
	}
	defer f.Close()
	header := make([]byte, 20)
	if _, e = io.ReadFull(f, header); e != nil || string(header[:4]) != "\x7fELF" || header[5] != 1 {
		return "", errors.New("not a little-endian ELF executable")
	}
	// e_machine is a 16-bit little-endian integer at byte 18.
	switch uint16(header[18]) | uint16(header[19])<<8 {
	case 62:
		return "x86_64", nil
	case 183:
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported ELF architecture")
	}
}
func parseVersion(s string) string {
	for _, f := range strings.Fields(s) {
		f = strings.TrimPrefix(f, "v")
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' {
			return f
		}
	}
	return ""
}
func safeVersion(v string) string {
	if v == "" || strings.ContainsAny(v, "/\\\x00") || v == "." || v == ".." {
		return ""
	}
	return v
}
func samePath(a, b string) bool { return a != "" && filepath.Clean(a) == filepath.Clean(b) }

var _ = runtime.GOARCH
