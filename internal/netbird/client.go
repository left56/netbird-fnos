package netbird

import (
	"context"
	"os/exec"
	"strings"
)

// Runner is used only for binary lifecycle probes (version, architecture and
// checksum validation). Runtime daemon control uses DaemonJSONClient.
type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, binary string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, binary, args...).Output()
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
	AllowServerSSH      bool   `json:"allowServerSSH"`
	BlockInbound        bool   `json:"blockInbound"`
	BlockLANAccess      bool   `json:"blockLANAccess"`
	DisableAutoConnect  bool   `json:"disableAutoConnect"`
	DisableClientRoutes bool   `json:"disableClientRoutes"`
}
type SSOLogin struct {
	VerificationURI string `json:"verificationURI"`
	UserCode        string
}

func safeValue(v string) bool  { return v != "" && len(v) <= 256 && !strings.ContainsAny(v, "\x00\r\n") }
func safeSecret(v string) bool { return len(v) <= 4096 && !strings.ContainsAny(v, "\x00\r\n") }
