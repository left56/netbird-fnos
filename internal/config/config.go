package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	defaultListenAddr = "127.0.0.1:8080"
	defaultTimeout    = 10 * time.Second
)

type Config struct {
	ListenAddr    string
	SocketPath    string
	GatewayPrefix string
	WebRoot       string
	PackageVar    string
	AppDest       string
	// Kept for P0 configuration compatibility; BinaryManager deliberately does
	// not use this user-controlled value.
	NetBirdBinary  string
	CommandTimeout time.Duration
	DaemonAddr     string
}

func Load() (Config, error) {
	cfg := Config{ListenAddr: value("NB_FNOS_LISTEN_ADDR", defaultListenAddr), SocketPath: os.Getenv("NB_FNOS_SOCKET"), GatewayPrefix: value("NB_FNOS_GATEWAY_PREFIX", "/"), WebRoot: os.Getenv("NB_FNOS_WEB_ROOT"), PackageVar: value("TRIM_PKGVAR", os.TempDir()), AppDest: value("TRIM_APPDEST", "."), NetBirdBinary: value("NB_FNOS_NETBIRD_BINARY", "netbird"), CommandTimeout: defaultTimeout}
	cfg.DaemonAddr = value("NB_FNOS_DAEMON_ADDR", "unix://"+filepath.Join(cfg.PackageVar, "netbird", "daemon.sock"))
	if raw := os.Getenv("NB_FNOS_COMMAND_TIMEOUT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("NB_FNOS_COMMAND_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.CommandTimeout = time.Duration(seconds) * time.Second
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
