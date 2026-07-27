package netbird

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// ProfileSettings is the only configuration surface exposed to the browser.
// Sensitive values are accepted only by write-specific APIs and never appear
// in this structure's JSON representation.
type ProfileSettings struct {
	Name                   string   `json:"name"`
	ManagementURL          string   `json:"managementURL"`
	InterfaceName          string   `json:"interfaceName"`
	InterfacePort          int      `json:"interfacePort"`
	MTU                    int      `json:"mtu"`
	DNS                    bool     `json:"dns"`
	IPv6                   bool     `json:"ipv6"`
	RouteExclusions        []string `json:"routeExclusions"`
	AllowSSH               bool     `json:"allowSSH"`
	BlockInbound           bool     `json:"blockInbound"`
	QuantumResistance      bool     `json:"quantumResistance"`
	RosenpassPermissive    bool     `json:"rosenpassPermissive"`
	ConnectOnStartup       bool     `json:"connectOnStartup"`
	LogLevel               string   `json:"logLevel"`
	LogFile                string   `json:"logFile"`
	WireGuardPort          int      `json:"wireGuardPort"`
	DisableClientRoutes    bool     `json:"disableClientRoutes"`
	DNSRouteInterval       int      `json:"dnsRouteInterval"`
	SetupKeyConfigured     bool     `json:"setupKeyConfigured"`
	PresharedKeyConfigured bool     `json:"presharedKeyConfigured"`
}
type ProfileConfigStore struct{ root string }

func NewProfileConfigStore(pkgvar string) *ProfileConfigStore {
	return &ProfileConfigStore{root: filepath.Join(pkgvar, "netbird", "profiles")}
}
func (s *ProfileConfigStore) Get(id string) (ProfileSettings, error) {
	if !safeProfileID(id) {
		return ProfileSettings{}, errors.New("invalid profile id")
	}
	raw, e := os.ReadFile(filepath.Join(s.root, id+".json"))
	if errors.Is(e, os.ErrNotExist) && id == "default" {
		return ProfileSettings{Name: "default"}, nil
	}
	if e != nil {
		return ProfileSettings{}, e
	}
	var rawMap map[string]json.RawMessage
	if e = json.Unmarshal(raw, &rawMap); e != nil {
		return ProfileSettings{}, e
	}
	var value ProfileSettings
	_ = json.Unmarshal(raw, &value)
	return value, nil
}
func (s *ProfileConfigStore) Put(id string, value ProfileSettings) error {
	if !safeProfileID(id) || value.Name == "" || value.InterfacePort < 0 || value.InterfacePort > 65535 || value.MTU < 0 || value.MTU > 10000 {
		return errors.New("invalid profile settings")
	}
	if e := os.MkdirAll(s.root, 0700); e != nil {
		return e
	}
	path := filepath.Join(s.root, id+".json")
	old := map[string]json.RawMessage{}
	if raw, e := os.ReadFile(path); e == nil {
		_ = json.Unmarshal(raw, &old)
		_ = os.WriteFile(path+".bak", raw, 0600)
	}
	known, _ := json.Marshal(value)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(known, &fields)
	for k, v := range fields {
		old[k] = v
	}
	raw, e := json.Marshal(old)
	if e != nil {
		return e
	}
	tmp, e := os.OpenFile(path+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	if _, e = tmp.Write(raw); e == nil {
		e = tmp.Sync()
	}
	if closeErr := tmp.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	if e = os.Rename(path+".tmp", path); e != nil {
		return e
	}
	_, e = s.Get(id)
	return e
}
func safeProfileID(id string) bool {
	return id != "" && len(id) <= 128 && filepath.Base(id) == id && id != "." && id != ".."
}
