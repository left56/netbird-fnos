package netbird

import (
	"context"
	"testing"
)

type runtimeFake struct{}

func (runtimeFake) Command(context.Context, ...string) ([]byte, error) {
	return []byte(`{"connected":true,"peers":{"one":{"fqdn":"peer","connectionStatus":"Connected","connectionType":"P2P"}}}`), nil
}
func (runtimeFake) Status(context.Context) Status { return Status{Connected: true} }
func (runtimeFake) Profiles(context.Context) ([]Profile, error) {
	return []Profile{{ID: "default", Name: "default", Active: true}}, nil
}
func (runtimeFake) Networks(context.Context) ([]Network, error) {
	return []Network{{ID: "n1", Name: "office", Selected: true}, {ID: "n2", Name: "egress", ExitNode: true}}, nil
}
func (runtimeFake) SelectNetworks(context.Context, []string, bool) error { return nil }
func (runtimeFake) DeselectNetworks(context.Context, []string) error     { return nil }

func TestRuntimeServicesReturnStructuredViews(t *testing.T) {
	status, err := NewStatusService(runtimeFake{}, nil, "0.2.0").Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Connection.Connected || status.Statistics.DirectPeers != 1 || status.Profile.Name != "default" {
		t.Fatalf("unexpected status: %#v", status)
	}
	peers, err := NewPeerService(runtimeFake{}).List(context.Background())
	if err != nil || len(peers) != 1 || peers[0].Name != "peer" {
		t.Fatalf("unexpected peers: %#v, %v", peers, err)
	}
	networks, err := NewNetworkService(runtimeFake{}).List(context.Background())
	if err != nil || len(networks.Selected) != 1 || len(networks.ExitNodes) != 1 {
		t.Fatalf("unexpected networks: %#v, %v", networks, err)
	}
}
