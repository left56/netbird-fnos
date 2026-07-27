package netbird

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DaemonJSONClient is the package's only runtime transport. It talks to the
// documented NetBird v0.75+ HTTP/JSON gateway over a package-private Unix
// socket. Browser requests never choose the socket address or RPC method.
type DaemonJSONClient struct {
	socket  string
	timeout time.Duration
	client  *http.Client
}

func NewDaemonJSONClient(address string, timeout time.Duration) (*DaemonJSONClient, error) {
	const prefix = "unix://"
	if !strings.HasPrefix(address, prefix) || len(strings.TrimPrefix(address, prefix)) == 0 {
		return nil, errors.New("NetBird daemon JSON address must be a Unix socket")
	}
	socket := strings.TrimPrefix(address, prefix)
	dialer := &net.Dialer{}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, "unix", socket)
	}}
	return &DaemonJSONClient{socket: socket, timeout: timeout, client: &http.Client{Transport: transport}}, nil
}

func (c *DaemonJSONClient) call(ctx context.Context, method string, request any, response any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/daemon.DaemonService/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return fmt.Errorf("NetBird daemon %s failed: %s", method, res.Status)
	}
	if response == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(response)
}

type daemonStatus struct {
	Status        string `json:"status"`
	DaemonVersion string `json:"daemonVersion"`
	FullStatus    struct {
		ManagementState struct {
			Connected bool   `json:"connected"`
			URL       string `json:"URL"`
		} `json:"managementState"`
		SignalState struct {
			Connected bool   `json:"connected"`
			URL       string `json:"URL"`
		} `json:"signalState"`
		LocalPeerState struct {
			IP     string `json:"IP"`
			PubKey string `json:"pubKey"`
			FQDN   string `json:"fqdn"`
		} `json:"localPeerState"`
		Peers []struct {
			IP             string   `json:"IP"`
			PubKey         string   `json:"pubKey"`
			ConnStatus     string   `json:"connStatus"`
			Relayed        bool     `json:"relayed"`
			FQDN           string   `json:"fqdn"`
			RemoteEndpoint string   `json:"remoteIceCandidateEndpoint"`
			Networks       []string `json:"networks"`
		} `json:"peers"`
	} `json:"fullStatus"`
}

func (c *DaemonJSONClient) daemonStatus(ctx context.Context) (daemonStatus, error) {
	var status daemonStatus
	err := c.call(ctx, "Status", map[string]any{"getFullPeerStatus": true}, &status)
	return status, err
}

func (c *DaemonJSONClient) Status(ctx context.Context) Status {
	v, err := c.daemonStatus(ctx)
	if err != nil {
		return Status{State: "unavailable", Detail: "NetBird daemon JSON socket is unavailable"}
	}
	return Status{State: first(v.Status, "unknown"), Connected: v.FullStatus.ManagementState.Connected}
}

// StatusJSON returns a normalized internal status representation for the
// runtime services. It never accepts a caller-provided RPC method.
func (c *DaemonJSONClient) StatusJSON(ctx context.Context) ([]byte, error) {
	v, err := c.daemonStatus(ctx)
	if err != nil {
		return nil, err
	}
	peers := map[string]map[string]any{}
	for _, p := range v.FullStatus.Peers {
		kind := "P2P"
		if p.Relayed {
			kind = "Relayed"
		}
		peers[p.PubKey] = map[string]any{"fqdn": p.FQDN, "ip": p.IP, "connectionStatus": p.ConnStatus, "connectionType": kind, "endpoint": p.RemoteEndpoint, "networks": p.Networks}
	}
	return json.Marshal(map[string]any{"connected": v.FullStatus.ManagementState.Connected, "status": v.Status, "managementState": connectedState(v.FullStatus.ManagementState.Connected), "signalState": connectedState(v.FullStatus.SignalState.Connected), "fqdn": v.FullStatus.LocalPeerState.FQDN, "netbirdIp": v.FullStatus.LocalPeerState.IP, "publicKey": v.FullStatus.LocalPeerState.PubKey, "peers": peers})
}

func connectedState(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

func (c *DaemonJSONClient) Connect(ctx context.Context, options ConnectOptions) error {
	if options.SetupKey != "" || options.ManagementURL != "" {
		if !safeSecret(options.SetupKey) || (options.ManagementURL != "" && !safeValue(options.ManagementURL)) {
			return errors.New("invalid connection settings")
		}
		login := map[string]any{"setupKey": options.SetupKey, "managementUrl": options.ManagementURL, "serverSSHAllowed": options.AllowServerSSH, "blockInbound": options.BlockInbound, "blockLanAccess": options.BlockLANAccess, "disableAutoConnect": options.DisableAutoConnect, "disableClientRoutes": options.DisableClientRoutes}
		if err := c.call(ctx, "Login", login, &map[string]any{}); err != nil {
			return err
		}
	}
	return c.call(ctx, "Up", map[string]any{}, &map[string]any{})
}
func (c *DaemonJSONClient) Disconnect(ctx context.Context) error {
	return c.call(ctx, "Down", map[string]any{}, &map[string]any{})
}
func (c *DaemonJSONClient) BeginSSO(ctx context.Context, managementURL string) (SSOLogin, error) {
	if !safeValue(managementURL) {
		return SSOLogin{}, errors.New("invalid management URL")
	}
	var out struct {
		Needs    bool   `json:"needsSSOLogin"`
		UserCode string `json:"userCode"`
		URL      string `json:"verificationURIComplete"`
	}
	if err := c.call(ctx, "Login", map[string]any{"managementUrl": managementURL}, &out); err != nil {
		return SSOLogin{}, err
	}
	if !out.Needs || out.UserCode == "" || out.URL == "" {
		return SSOLogin{}, errors.New("SSO login unavailable")
	}
	return SSOLogin{VerificationURI: out.URL, UserCode: out.UserCode}, nil
}
func (c *DaemonJSONClient) WaitSSO(ctx context.Context, code string) error {
	if !safeSecret(code) || code == "" {
		return errors.New("invalid SSO code")
	}
	if err := c.call(ctx, "WaitSSOLogin", map[string]any{"userCode": code}, &map[string]any{}); err != nil {
		return err
	}
	return c.call(ctx, "Up", map[string]any{}, &map[string]any{})
}

func (c *DaemonJSONClient) Profiles(ctx context.Context) ([]Profile, error) {
	var out struct {
		Profiles []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Active bool   `json:"isActive"`
		} `json:"profiles"`
	}
	if err := c.call(ctx, "ListProfiles", map[string]any{}, &out); err != nil {
		return nil, err
	}
	status := c.Status(ctx)
	result := make([]Profile, 0, len(out.Profiles))
	for _, p := range out.Profiles {
		result = append(result, Profile{ID: p.ID, Name: p.Name, Active: p.Active, Default: strings.EqualFold(p.Name, "default"), Connected: p.Active && status.Connected})
	}
	return result, nil
}
func (c *DaemonJSONClient) AddProfile(ctx context.Context, name string) error {
	if !safeValue(name) {
		return errors.New("invalid profile name")
	}
	return c.call(ctx, "AddProfile", map[string]any{"profileName": name}, &map[string]any{})
}
func (c *DaemonJSONClient) SelectProfile(ctx context.Context, handle string) error {
	if !safeValue(handle) {
		return errors.New("invalid profile handle")
	}
	return c.call(ctx, "SwitchProfile", map[string]any{"profileName": handle}, &map[string]any{})
}
func (c *DaemonJSONClient) RenameProfile(ctx context.Context, handle, name string) error {
	if !safeValue(handle) || !safeValue(name) {
		return errors.New("invalid profile value")
	}
	return c.call(ctx, "RenameProfile", map[string]any{"handle": handle, "newProfileName": name}, &map[string]any{})
}
func (c *DaemonJSONClient) RemoveProfile(ctx context.Context, handle string) error {
	if !safeValue(handle) {
		return errors.New("invalid profile handle")
	}
	return c.call(ctx, "RemoveProfile", map[string]any{"profileName": handle}, &map[string]any{})
}

func (c *DaemonJSONClient) Networks(ctx context.Context) ([]Network, error) {
	var out struct {
		Routes []struct {
			ID       string   `json:"ID"`
			Range    string   `json:"range"`
			Selected bool     `json:"selected"`
			Domains  []string `json:"domains"`
		} `json:"routes"`
	}
	if err := c.call(ctx, "ListNetworks", map[string]any{}, &out); err != nil {
		return nil, err
	}
	result := make([]Network, 0, len(out.Routes))
	for _, v := range out.Routes {
		name := v.Range
		if name == "" || strings.EqualFold(name, "invalid Prefix") {
			name = strings.Join(v.Domains, ", ")
		}
		if name == "" {
			name = "未命名 Network"
		}
		result = append(result, Network{ID: v.ID, Name: name, Selected: v.Selected, Domains: v.Domains})
	}
	return result, nil
}
func (c *DaemonJSONClient) SelectNetworks(ctx context.Context, ids []string, appendMode bool) error {
	if len(ids) == 0 || len(ids) > 64 {
		return errors.New("invalid network selection")
	}
	for _, id := range ids {
		if !safeValue(id) {
			return errors.New("invalid network identifier")
		}
	}
	return c.call(ctx, "SelectNetworks", map[string]any{"networkIDs": ids, "append": appendMode}, &map[string]any{})
}
func (c *DaemonJSONClient) DeselectNetworks(ctx context.Context, ids []string) error {
	if len(ids) == 0 || len(ids) > 64 {
		return errors.New("invalid network selection")
	}
	for _, id := range ids {
		if !safeValue(id) {
			return errors.New("invalid network identifier")
		}
	}
	return c.call(ctx, "DeselectNetworks", map[string]any{"networkIDs": ids}, &map[string]any{})
}
