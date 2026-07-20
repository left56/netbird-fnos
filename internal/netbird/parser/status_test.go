package parser

import "testing"

func TestStatusJSONProducesSafePeerView(t *testing.T) {
	status, err := StatusJSON([]byte(`{"connected":true,"managementState":"Connected","signalState":"Connected","netbirdIp":"100.64.0.2","peers":{"p1":{"fqdn":"workstation","ip":"100.64.0.3","connectionStatus":"Connected","connectionType":"P2P","endpoint":"198.51.100.2:51820"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !status.Connected || status.IP != "100.64.0.2" || len(status.Peers) != 1 {
		t.Fatalf("unexpected status: %#v", status)
	}
	if !status.Peers[0].Direct || !status.Peers[0].Connected || status.Peers[0].ID != "p1" {
		t.Fatalf("unexpected peer: %#v", status.Peers[0])
	}
}

func TestNetworksTextRecognizesSelectionAndExitNode(t *testing.T) {
	got := NetworksText("ID Name State\nnet1 office ✓ selected\nnet2 exit exit-node")
	if len(got) != 2 || !got[0].Selected || !got[1].ExitNode {
		t.Fatalf("unexpected networks: %#v", got)
	}
}
