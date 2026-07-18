package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	defaultListenAddr = "127.0.0.1:8080"
	defaultBinary     = "netbird"
	defaultTimeout    = 10 * time.Second
)

type Config struct {
	ListenAddr     string
	SocketPath     string
	GatewayPrefix  string
	WebRoot        string
	NetBirdBinary  string
	CommandTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{ListenAddr: value("NB_FNOS_LISTEN_ADDR", defaultListenAddr), SocketPath: os.Getenv("NB_FNOS_SOCKET"), GatewayPrefix: value("NB_FNOS_GATEWAY_PREFIX", "/"), WebRoot: os.Getenv("NB_FNOS_WEB_ROOT"), NetBirdBinary: value("NB_FNOS_NETBIRD_BINARY", defaultBinary), CommandTimeout: defaultTimeout}
	if raw := os.Getenv("NB_FNOS_COMMAND_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("NB_FNOS_COMMAND_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.CommandTimeout = time.Duration(seconds) * time.Second
	}
	if cfg.NetBirdBinary == "" {
		return Config{}, fmt.Errorf("NB_FNOS_NETBIRD_BINARY must not be empty")
	}
	if cfg.SocketPath == "" {
		if err := loopbackAddress(cfg.ListenAddr); err != nil {
			return Config{}, err
		}
	}
	if cfg.GatewayPrefix == "" || cfg.GatewayPrefix[0] != '/' {
		return Config{}, fmt.Errorf("NB_FNOS_GATEWAY_PREFIX must start with /")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loopbackAddress(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid NB_FNOS_LISTEN_ADDR: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("NB_FNOS_LISTEN_ADDR must use a loopback address")
	}
	return nil
}
