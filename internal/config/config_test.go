package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("NB_FNOS_LISTEN_ADDR", "")
	t.Setenv("NB_FNOS_NETBIRD_BINARY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" || cfg.NetBirdBinary != "netbird" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestLoadRejectsPublicListener(t *testing.T) {
	t.Setenv("NB_FNOS_LISTEN_ADDR", "0.0.0.0:8080")
	if _, err := Load(); err == nil {
		t.Fatal("expected loopback validation error")
	}
}
