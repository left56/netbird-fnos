package netbird

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/left56/netbird-fnos/internal/netbird/parser"
)

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, args...).Output()
}

type Client struct {
	runner     Runner
	binary     func(context.Context) string
	timeout    time.Duration
	daemonAddr string
}
type Status struct {
	State     string `json:"state"`
	Connected bool   `json:"connected"`
	Detail    string `json:"detail,omitempty"`
}
type Profile struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Active          bool   `json:"active"`
	Default         bool   `json:"default"`
	Connected       bool   `json:"connected"`
	ManagementURL   string `json:"managementURL,omitempty"`
	NetBirdIP       string `json:"netbirdIP,omitempty"`
	EnabledNetworks int    `json:"enabledNetworks"`
	ExitNode        string `json:"exitNode,omitempty"`
	LastConnectedAt string `json:"lastConnectedAt,omitempty"`
}
type Network struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Selected    bool   `json:"selected"`
	ExitNode    bool   `json:"exitNode"`
	Overlapping bool   `json:"overlap"`
}
type ConnectOptions struct {
	ManagementURL       string `json:"managementURL"`
	SetupKey            string `json:"setupKey"`
	AllowServerSSH      bool `json:"allowServerSSH"`
	BlockInbound        bool `json:"blockInbound"`
	BlockLANAccess      bool `json:"blockLANAccess"`
	DisableAutoConnect  bool `json:"disableAutoConnect"`
	DisableClientRoutes bool `json:"disableClientRoutes"`
}

func NewClient(runner Runner, binary string, timeout time.Duration) Client {
	return Client{runner: runner, binary: func(context.Context) string { return binary }, timeout: timeout}
}

func NewManagedClient(runner Runner, manager *BinaryManager, timeout time.Duration) Client {
	return Client{runner: runner, binary: manager.Path, timeout: timeout}
}

// NewManagedClientWithDaemon binds runtime commands to the package-owned
// daemon socket. The socket address is supplied by package lifecycle code,
// never by a browser request.
func NewManagedClientWithDaemon(runner Runner, manager *BinaryManager, timeout time.Duration, daemonAddr string) Client {
	return Client{runner: runner, binary: manager.Path, timeout: timeout, daemonAddr: daemonAddr}
}

func (c Client) Status(ctx context.Context) Status {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	output, err := c.run(ctx, "status", "--json")
	if err != nil {
		return unavailable(err)
	}
	var response struct {
		Status    string `json:"status"`
		Connected bool   `json:"connected"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return Status{State: "unavailable", Detail: "official NetBird CLI returned an unsupported response"}
	}
	if response.Status == "" {
		response.Status = "unknown"
	}
	return Status{State: response.Status, Connected: response.Connected}
}

func (c Client) Connect(ctx context.Context, options ConnectOptions) error {
	args := []string{"up"}
	if options.ManagementURL != "" {
		if !safeValue(options.ManagementURL) {
			return errors.New("invalid management URL")
		}
		args = append(args, "--management-url", options.ManagementURL)
	}
	if options.SetupKey != "" {
		if !safeSecret(options.SetupKey) {
			return errors.New("invalid setup key")
		}
		args = append(args, "--setup-key", options.SetupKey)
	}
	if options.AllowServerSSH {
		args = append(args, "--allow-server-ssh")
	}
	if options.BlockInbound {
		args = append(args, "--block-inbound")
	}
	if options.BlockLANAccess {
		args = append(args, "--block-lan-access")
	}
	if options.DisableAutoConnect {
		args = append(args, "--disable-auto-connect")
	}
	if options.DisableClientRoutes {
		args = append(args, "--disable-client-routes")
	}
	_, err := c.run(ctx, args...)
	return err
}
func (c Client) Disconnect(ctx context.Context) error { _, err := c.run(ctx, "down"); return err }
func (c Client) Profiles(ctx context.Context) ([]Profile, error) {
	out, err := c.run(ctx, "profile", "list", "--show-id")
	if err != nil {
		// A daemon can temporarily reject profile RPCs while it is starting.
		// Preserve the normal default profile view whenever the controlled CLI
		// still reports a usable client instead of turning the page into an
		// unsupported-feature error.
		status := c.Status(ctx)
		if status.State != "unavailable" {
			return []Profile{{ID: "default", Name: "default", Default: true, Active: true, Connected: status.Connected}}, nil
		}
		return nil, err
	}
	profiles := parseProfiles(string(out))
	// NetBird guarantees a default profile for profile-capable clients. Empty
	// table output is observed before the first persisted profile is rendered;
	// verify the controlled CLI status and report that real default identity.
	if len(profiles) == 0 {
		status := c.Status(ctx)
		if status.State != "unavailable" {
			return []Profile{{ID: "default", Name: "default", Default: true, Active: true, Connected: status.Connected}}, nil
		}
	}
	for i := range profiles {
		profiles[i].Default = profiles[i].ID == "default" || strings.EqualFold(profiles[i].Name, "default")
		profiles[i].Connected = profiles[i].Active && c.Status(ctx).Connected
	}
	return profiles, nil
}
func (c Client) AddProfile(ctx context.Context, name string) error {
	if !safeValue(name) {
		return errors.New("invalid profile name")
	}
	_, err := c.run(ctx, "profile", "add", name)
	return err
}
func (c Client) SelectProfile(ctx context.Context, handle string) error {
	if !safeValue(handle) {
		return errors.New("invalid profile handle")
	}
	_, err := c.run(ctx, "profile", "select", handle)
	return err
}
func (c Client) RenameProfile(ctx context.Context, handle, name string) error {
	if !safeValue(handle) || !safeValue(name) {
		return errors.New("invalid profile value")
	}
	_, err := c.run(ctx, "profile", "rename", handle, name)
	return err
}
func (c Client) RemoveProfile(ctx context.Context, handle string) error {
	if !safeValue(handle) {
		return errors.New("invalid profile handle")
	}
	_, err := c.run(ctx, "profile", "remove", handle)
	return err
}
func (c Client) Networks(ctx context.Context) ([]Network, error) {
	out, err := c.run(ctx, "networks", "list")
	if err != nil {
		return nil, err
	}
	parsed := parser.NetworksText(string(out))
	result := make([]Network, 0, len(parsed))
	for _, n := range parsed {
		result = append(result, Network{ID: n.ID, Name: n.Name, Selected: n.Selected, ExitNode: n.ExitNode, Overlapping: n.Overlapping})
	}
	return result, nil
}
func (c Client) SelectNetworks(ctx context.Context, ids []string, appendMode bool) error {
	if len(ids) == 0 || len(ids) > 64 {
		return errors.New("invalid network selection")
	}
	args := []string{"networks", "select"}
	if appendMode {
		args = append(args, "--append")
	}
	for _, id := range ids {
		if !safeValue(id) {
			return errors.New("invalid network identifier")
		}
		args = append(args, id)
	}
	_, err := c.run(ctx, args...)
	return err
}
func (c Client) DeselectNetworks(ctx context.Context, ids []string) error {
	if len(ids) == 0 || len(ids) > 64 {
		return errors.New("invalid network selection")
	}
	args := []string{"networks", "deselect"}
	for _, id := range ids {
		if !safeValue(id) {
			return errors.New("invalid network identifier")
		}
		args = append(args, id)
	}
	_, err := c.run(ctx, args...)
	return err
}
func (c Client) run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	binary := c.binary(ctx)
	if binary == "" {
		return nil, exec.ErrNotFound
	}
	if c.daemonAddr != "" {
		args = append([]string{"--daemon-addr", c.daemonAddr}, args...)
	}
	return c.runner.Run(ctx, binary, args...)
}
func safeValue(v string) bool { return v != "" && len(v) <= 256 && !strings.ContainsAny(v, "\x00\r\n") }
func safeSecret(v string) bool { return len(v) <= 4096 && !strings.ContainsAny(v, "\x00\r\n") }
func parseProfiles(out string) []Profile {
	var result []Profile
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.EqualFold(fields[0], "ID") {
			continue
		}
		p := Profile{ID: fields[0], Name: strings.Join(fields[1:len(fields)-1], " "), Default: fields[0] == "default"}
		if len(fields) == 2 {
			p.Name = fields[1]
		}
		p.Active = strings.Contains(line, "✓") || strings.Contains(strings.ToLower(line), "active")
		p.Default = p.Default || strings.EqualFold(p.Name, "default")
		result = append(result, p)
	}
	return result
}

func unavailable(err error) Status {
	if errors.Is(err, exec.ErrNotFound) {
		return Status{State: "unavailable", Detail: "official NetBird CLI is not installed"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Status{State: "unavailable", Detail: "official NetBird CLI timed out"}
	}
	return Status{State: "unavailable", Detail: "official NetBird CLI is unavailable"}
}

func (c Client) String() string {
	return fmt.Sprintf("netbird client (%s)", c.binary(context.Background()))
}
