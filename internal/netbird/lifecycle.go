package netbird

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxUploadSize = 100 << 20

type DownloadSource struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Enabled        bool   `json:"enabled"`
	Priority       int    `json:"priority"`
	URLTemplate    string `json:"urlTemplate"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}
type Release struct {
	Version  string `json:"version"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
	Arch     string `json:"arch"`
}
type Lifecycle struct {
	manager *BinaryManager
	root    string
	client  *http.Client
	mu      sync.Mutex
}

func NewLifecycle(manager *BinaryManager, pkgvar string) *Lifecycle {
	return &Lifecycle{manager: manager, root: filepath.Join(pkgvar, "netbird"), client: &http.Client{Timeout: 60 * time.Second}}
}
func (l *Lifecycle) Check(ctx context.Context, version, arch string) (Release, error) {
	return l.release(ctx, version, arch)
}
func (l *Lifecycle) Download(ctx context.Context, version, arch, sourceID string) (Binary, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if version == "" {
		return Binary{}, errors.New("a specific version is required for installation")
	}
	r, err := l.release(ctx, version, arch)
	if err != nil {
		return Binary{}, err
	}
	source, err := l.source(sourceID)
	if err != nil {
		return Binary{}, err
	}
	u := r.URL
	if source != nil {
		u, err = rewriteURL(source.URLTemplate, r.URL)
		if err != nil {
			return Binary{}, err
		}
	}
	staged, err := l.fetch(ctx, u, source, r.Checksum)
	if err != nil {
		return Binary{}, err
	}
	defer os.Remove(staged)
	return l.manager.Install(ctx, staged, "official", r.URL, true)
}
func (l *Lifecycle) Upload(ctx context.Context, name string, body io.Reader, allowUnverified bool) (Binary, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	staged, err := l.saveUpload(body)
	if err != nil {
		return Binary{}, err
	}
	defer os.RemoveAll(filepath.Dir(staged))
	binary, err := extractUpload(staged, name)
	if err != nil {
		return Binary{}, err
	}
	if binary != staged {
		defer os.Remove(binary)
	}
	return l.manager.Install(ctx, binary, "upload-unverified", "", allowUnverified)
}
func (l *Lifecycle) Sources() ([]DownloadSource, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.readSources()
}
func (l *Lifecycle) SaveSource(s DownloadSource) (DownloadSource, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if s.ID == "" {
		s.ID = fmt.Sprintf("source-%d", time.Now().UnixNano())
	}
	if s.Name == "" || s.TimeoutSeconds < 1 || s.TimeoutSeconds > 300 {
		return DownloadSource{}, errors.New("invalid download source")
	}
	if _, err := rewriteURL(s.URLTemplate, "https://github.com/netbirdio/netbird/releases/download/v0.0.0/netbird_0.0.0_linux_amd64.tar.gz"); err != nil {
		return DownloadSource{}, err
	}
	all, err := l.readSources()
	if err != nil {
		return DownloadSource{}, err
	}
	found := false
	for i := range all {
		if all[i].ID == s.ID {
			all[i] = s
			found = true
		}
	}
	if !found {
		all = append(all, s)
	}
	return s, l.writeSources(all)
}
func (l *Lifecycle) DeleteSource(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	all, err := l.readSources()
	if err != nil {
		return err
	}
	out := all[:0]
	for _, s := range all {
		if s.ID != id {
			out = append(out, s)
		}
	}
	if len(out) == len(all) {
		return os.ErrNotExist
	}
	return l.writeSources(out)
}
func (l *Lifecycle) TestSource(ctx context.Context, id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, err := l.source(id)
	if err != nil {
		return err
	}
	if s == nil {
		return errors.New("source not found")
	}
	u, err := rewriteURL(s.URLTemplate, "https://github.com/")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u, nil)
	if err != nil {
		return err
	}
	c := *l.client
	c.Timeout = time.Duration(s.TimeoutSeconds) * time.Second
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("source returned %d", resp.StatusCode)
	}
	return nil
}
func (l *Lifecycle) release(ctx context.Context, version, arch string) (Release, error) {
	if arch == "" {
		arch = hostArch()
	}
	relArch := map[string]string{"x86_64": "amd64", "arm64": "arm64"}[arch]
	if relArch == "" {
		return Release{}, errors.New("unsupported architecture")
	}
	endpoint := "https://api.github.com/repos/netbirdio/netbird/releases/latest"
	if version != "" {
		endpoint = "https://api.github.com/repos/netbirdio/netbird/releases/tags/v" + strings.TrimPrefix(version, "v")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Release{}, fmt.Errorf("official release lookup returned %d", resp.StatusCode)
	}
	var payload struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return Release{}, err
	}
	v := strings.TrimPrefix(version, "v")
	if v == "" {
		v = strings.TrimPrefix(payload.Tag, "v")
	}
	if v == "" {
		return Release{}, errors.New("official release has no version")
	}
	asset := "netbird_" + v + "_linux_" + relArch + ".tar.gz"
	checksums := "netbird_" + v + "_checksums.txt"
	var au, cu string
	for _, a := range payload.Assets {
		if a.Name == asset {
			au = a.URL
		}
		if a.Name == checksums {
			cu = a.URL
		}
	}
	if au == "" || cu == "" {
		return Release{}, errors.New("official release assets are incomplete")
	}
	sum, err := l.checksumFile(ctx, cu, asset)
	if err != nil {
		return Release{}, err
	}
	return Release{Version: v, URL: au, Checksum: sum, Arch: arch}, nil
}
func (l *Lifecycle) checksumFile(ctx context.Context, u, name string) (string, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if e != nil {
		return "", e
	}
	resp, e := l.client.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("cannot download official checksums")
	}
	raw, e := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if e != nil {
		return "", e
	}
	re := regexp.MustCompile(`(?m)^([a-fA-F0-9]{64})\s+\*?` + regexp.QuoteMeta(name) + `\s*$`)
	m := re.FindSubmatch(raw)
	if m == nil {
		return "", errors.New("official checksum is missing")
	}
	return strings.ToLower(string(m[1])), nil
}
func (l *Lifecycle) fetch(ctx context.Context, u string, s *DownloadSource, expected string) (string, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if e != nil {
		return "", e
	}
	c := *l.client
	if s != nil {
		c.Timeout = time.Duration(s.TimeoutSeconds) * time.Second
	}
	resp, e := c.Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %d", resp.StatusCode)
	}
	if e = os.MkdirAll(filepath.Join(l.root, "downloads"), 0755); e != nil {
		return "", e
	}
	f, e := os.CreateTemp(filepath.Join(l.root, "downloads"), "release-*")
	if e != nil {
		return "", e
	}
	defer func() {
		if e != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	h := sha256.New()
	if _, e = io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, maxUploadSize+1)); e != nil {
		return "", e
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	info, e := os.Stat(f.Name())
	if e != nil || info.Size() > maxUploadSize {
		return "", errors.New("download exceeds size limit")
	}
	if fmt.Sprintf("%x", h.Sum(nil)) != expected {
		return "", errors.New("official checksum mismatch")
	}
	binary, e := extractUpload(f.Name(), "release.tar.gz")
	if e != nil {
		return "", e
	}
	os.Remove(f.Name())
	return binary, nil
}
func (l *Lifecycle) saveUpload(body io.Reader) (string, error) {
	if err := os.MkdirAll(filepath.Join(l.root, "downloads"), 0755); err != nil {
		return "", err
	}
	d, err := os.MkdirTemp(filepath.Join(l.root, "downloads"), "upload-")
	if err != nil {
		return "", err
	}
	f, err := os.Create(filepath.Join(d, "input"))
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(body, maxUploadSize+1))
	closeErr := f.Close()
	if err != nil || closeErr != nil || n > maxUploadSize {
		os.RemoveAll(d)
		return "", errors.New("upload exceeds size limit")
	}
	return f.Name(), nil
}
func (l *Lifecycle) readSources() ([]DownloadSource, error) {
	raw, e := os.ReadFile(filepath.Join(l.root, "sources.json"))
	if errors.Is(e, os.ErrNotExist) {
		return []DownloadSource{}, nil
	}
	if e != nil {
		return nil, e
	}
	var all []DownloadSource
	e = json.Unmarshal(raw, &all)
	sort.SliceStable(all, func(i, j int) bool { return all[i].Priority < all[j].Priority })
	return all, e
}
func (l *Lifecycle) writeSources(all []DownloadSource) error {
	if e := os.MkdirAll(l.root, 0755); e != nil {
		return e
	}
	raw, e := json.Marshal(all)
	if e != nil {
		return e
	}
	tmp := filepath.Join(l.root, ".sources.tmp")
	if e = os.WriteFile(tmp, raw, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, filepath.Join(l.root, "sources.json"))
}
func (l *Lifecycle) source(id string) (*DownloadSource, error) {
	if id == "" {
		return nil, nil
	}
	all, e := l.readSources()
	if e != nil {
		return nil, e
	}
	for _, s := range all {
		if s.ID == id && s.Enabled {
			return &s, nil
		}
	}
	return nil, errors.New("download source not found")
}
func rewriteURL(template, official string) (string, error) {
	if strings.Count(template, "{url}") != 1 {
		return "", errors.New("URL template must contain exactly one {url}")
	}
	u := strings.ReplaceAll(template, "{url}", url.QueryEscape(official))
	parsed, e := url.Parse(u)
	if e != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errors.New("download source must use HTTPS")
	}
	return u, nil
}
func extractUpload(input, name string) (string, error) {
	f, e := os.Open(input)
	if e != nil {
		return "", e
	}
	defer f.Close()
	head := make([]byte, 2)
	_, _ = io.ReadFull(f, head)
	if _, e = f.Seek(0, 0); e != nil {
		return "", e
	}
	if string(head) != "\x1f\x8b" {
		if _, e = elfArch(input); e != nil {
			return "", errors.New("upload must be an ELF binary or gzip archive")
		}
		return input, nil
	}
	gz, e := gzip.NewReader(f)
	if e != nil {
		return "", e
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out, e := os.CreateTemp(filepath.Dir(input), "netbird-")
	if e != nil {
		return "", e
	}
	outPath := out.Name()
	found := false
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			out.Close()
			os.Remove(outPath)
			return "", e
		}
		if filepath.IsAbs(h.Name) || strings.Contains(filepath.Clean(h.Name), "..") || h.Typeflag != tar.TypeReg {
			out.Close()
			os.Remove(outPath)
			return "", errors.New("unsafe archive entry")
		}
		if filepath.Base(h.Name) != "netbird" {
			continue
		}
		if found {
			out.Close()
			os.Remove(outPath)
			return "", errors.New("archive contains multiple netbird files")
		}
		if _, e = io.Copy(out, io.LimitReader(tr, maxUploadSize)); e != nil {
			out.Close()
			os.Remove(outPath)
			return "", e
		}
		found = true
	}
	if e = out.Close(); e != nil {
		return "", e
	}
	if !found {
		os.Remove(outPath)
		return "", errors.New("archive has no netbird binary")
	}
	if e = os.Chmod(outPath, 0755); e != nil {
		return "", e
	}
	if _, e = elfArch(outPath); e != nil {
		os.Remove(outPath)
		return "", e
	}
	return outPath, nil
}
func hostArch() string { return map[string]string{"amd64": "x86_64", "arm64": "arm64"}[runtime.GOARCH] }
