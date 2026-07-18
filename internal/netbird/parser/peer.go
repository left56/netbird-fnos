package parser

// Peer is the safe, structured peer view exposed to the fnOS UI.
type Peer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	IP        string   `json:"ip"`
	Connected bool     `json:"connected"`
	Direct    bool     `json:"direct"`
	LatencyMS int      `json:"latencyMs,omitempty"`
	Endpoint  string   `json:"endpoint"`
	OS        string   `json:"os"`
	Version   string   `json:"version"`
	LastSeen  string   `json:"lastSeen,omitempty"`
	Networks  []string `json:"networks"`
}

func peersFromStatus(value map[string]any) []Peer {
	raw, ok := value["peers"].(map[string]any)
	if !ok {
		return []Peer{}
	}
	peers := make([]Peer, 0, len(raw))
	for id, entry := range raw {
		fields, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		peer := Peer{ID: id, Networks: []string{}}
		peer.Name, _ = fields["fqdn"].(string)
		peer.IP, _ = fields["ip"].(string)
		state, _ := fields["connectionStatus"].(string)
		peer.Connected = state == "Connected" || state == "connected"
		kind, _ := fields["connectionType"].(string)
		peer.Direct = kind == "P2P" || kind == "direct"
		peer.Endpoint, _ = fields["endpoint"].(string)
		peer.OS, _ = fields["os"].(string)
		peer.Version, _ = fields["version"].(string)
		peer.LastSeen, _ = fields["lastSeen"].(string)
		if latency, ok := fields["latency"].(float64); ok {
			peer.LatencyMS = int(latency)
		}
		peers = append(peers, peer)
	}
	return peers
}
