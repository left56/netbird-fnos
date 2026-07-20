package netbird

import (
	"context"
	"fmt"

	"github.com/left56/netbird-fnos/internal/netbird/parser"
)

type CommandRunner interface {
	Command(context.Context, ...string) ([]byte, error)
	Status(context.Context) Status
	Profiles(context.Context) ([]Profile, error)
	Networks(context.Context) ([]Network, error)
	SelectNetworks(context.Context, []string, bool) error
	DeselectNetworks(context.Context, []string) error
}

func (c Client) Command(ctx context.Context, args ...string) ([]byte, error) {
	return c.run(ctx, args...)
}

type StatusService struct {
	client  CommandRunner
	manager *BinaryManager
	wrapper string
}

// RuntimeStatus is the stable, UI-facing status representation. It deliberately
// does not expose the CLI's raw JSON because that is an implementation detail of
// NetBird and can contain fields which are not suitable for the browser.
type RuntimeStatus struct {
	Connection RuntimeConnection `json:"connection"`
	Profile    RuntimeProfile    `json:"profile"`
	Device     RuntimeDevice     `json:"device"`
	Statistics RuntimeStatistics `json:"statistics"`
	Versions   RuntimeVersions   `json:"versions"`
}
type RuntimeConnection struct {
	Connected  bool   `json:"connected"`
	Management string `json:"management"`
	Signal     string `json:"signal"`
	Relay      bool   `json:"relay"`
}
type RuntimeProfile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}
type RuntimeDevice struct {
	Hostname  string `json:"hostname"`
	NetBirdIP string `json:"netbirdIp"`
	PublicKey string `json:"publicKey"`
	Interface string `json:"interface"`
}
type RuntimeStatistics struct {
	PeerCount       int `json:"peerCount"`
	OnlinePeers     int `json:"onlinePeers"`
	DirectPeers     int `json:"directPeers"`
	RelayPeers      int `json:"relayPeers"`
	EnabledNetworks int `json:"enabledNetworks"`
}
type RuntimeVersions struct {
	Wrapper string `json:"wrapper"`
	NetBird string `json:"netbird"`
	Bundled string `json:"bundled"`
	Source  Source `json:"source"`
}

func NewStatusService(c CommandRunner, m *BinaryManager, w string) *StatusService {
	return &StatusService{c, m, w}
}
func (s *StatusService) Get(ctx context.Context) (RuntimeStatus, error) {
	raw, e := s.client.Command(ctx, "status", "--json")
	if e != nil {
		return RuntimeStatus{}, e
	}
	p, e := parser.StatusJSON(raw)
	if e != nil {
		return RuntimeStatus{}, e
	}
	profiles, _ := s.client.Profiles(ctx)
	var active Profile
	for _, v := range profiles {
		if v.Active {
			active = v
		}
	}
	networks, _ := s.client.Networks(ctx)
	online, direct := 0, 0
	for _, v := range p.Peers {
		if v.Connected {
			online++
		}
		if v.Direct {
			direct++
		}
	}
	b := Binary{Source: Missing}
	if s.manager != nil {
		b = s.manager.Resolve(ctx)
	}
	return RuntimeStatus{
		Connection: RuntimeConnection{Connected: p.Connected, Management: p.Management, Signal: p.Signal, Relay: online-direct > 0},
		Profile:    RuntimeProfile{ID: active.ID, Name: active.Name, Source: "cli"},
		Device:     RuntimeDevice{Hostname: p.Hostname, NetBirdIP: p.IP, PublicKey: p.PublicKey, Interface: p.Interface},
		Statistics: RuntimeStatistics{PeerCount: len(p.Peers), OnlinePeers: online, DirectPeers: direct, RelayPeers: online - direct, EnabledNetworks: len(networks)},
		Versions:   RuntimeVersions{Wrapper: s.wrapper, NetBird: b.Version, Source: b.Source},
	}, nil
}

type PeerService struct{ client CommandRunner }

func NewPeerService(c CommandRunner) *PeerService { return &PeerService{c} }
func (s *PeerService) List(ctx context.Context) ([]parser.Peer, error) {
	raw, e := s.client.Command(ctx, "status", "--json")
	if e != nil {
		return nil, e
	}
	v, e := parser.StatusJSON(raw)
	return v.Peers, e
}

type NetworkService struct{ client CommandRunner }

func NewNetworkService(c CommandRunner) *NetworkService { return &NetworkService{c} }

type NetworkList struct {
	All          []Network    `json:"all"`
	Overlapping  []Network    `json:"overlapping"`
	ExitNodes    []Network    `json:"exitNodes"`
	Selected     []Network    `json:"selected"`
	Pending      []Network    `json:"pending"`
	Capabilities Capabilities `json:"capabilities"`
}

func (s *NetworkService) List(ctx context.Context) (NetworkList, error) {
	v, e := s.client.Networks(ctx)
	if e != nil {
		return NetworkList{}, e
	}
	result := NetworkList{All: v, Overlapping: []Network{}, ExitNodes: []Network{}, Selected: []Network{}, Pending: []Network{}, Capabilities: Capabilities{Networks: true, ExitNode: true}}
	for _, n := range v {
		if n.Selected {
			result.Selected = append(result.Selected, n)
		}
		if n.ExitNode {
			result.ExitNodes = append(result.ExitNodes, n)
		}
	}
	return result, nil
}

func (s *NetworkService) Select(ctx context.Context, ids []string, appendMode bool) error {
	if len(ids) == 0 {
		return fmt.Errorf("network selection is empty")
	}
	return s.client.SelectNetworks(ctx, ids, appendMode)
}
func (s *NetworkService) Deselect(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("network selection is empty")
	}
	return s.client.DeselectNetworks(ctx, ids)
}
