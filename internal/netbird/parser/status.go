package parser

import "encoding/json"

type Status struct {
	Connected  bool
	Management string
	Signal     string
	Hostname   string
	IP         string
	PublicKey  string
	Interface  string
	Peers      []Peer
}

func StatusJSON(raw []byte) (Status, error) {
	var v map[string]any
	if e := json.Unmarshal(raw, &v); e != nil {
		return Status{}, e
	}
	s := Status{}
	s.Connected, _ = v["connected"].(bool)
	s.Management, _ = v["managementState"].(string)
	s.Signal, _ = v["signalState"].(string)
	s.Hostname, _ = v["fqdn"].(string)
	s.IP, _ = v["netbirdIp"].(string)
	s.PublicKey, _ = v["publicKey"].(string)
	s.Interface, _ = v["interface"].(string)
	s.Peers = peersFromStatus(v)
	return s, nil
}
