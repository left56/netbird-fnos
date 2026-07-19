package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/left56/netbird-fnos/internal/api"
	"github.com/left56/netbird-fnos/internal/config"
	"github.com/left56/netbird-fnos/internal/netbird"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	manager := netbird.NewBinaryManager(cfg.PackageVar, cfg.AppDest, netbird.ExecRunner{}, cfg.CommandTimeout)
	lifecycle := netbird.NewLifecycle(manager, cfg.PackageVar)
	client := netbird.NewManagedClientWithDaemon(netbird.ExecRunner{}, manager, cfg.CommandTimeout, cfg.DaemonAddr)
	profiles := netbird.NewProfileService(client, netbird.NewProfileConfigStore(cfg.PackageVar))
	status := netbird.NewStatusService(client, manager, version)
	peers := netbird.NewPeerService(client)
	networks := netbird.NewNetworkService(client)
	logs := api.NewLogReader(filepath.Join(cfg.PackageVar, "netbird-fnos-api.log"), filepath.Join(cfg.PackageVar, "netbird", "daemon.log"))
	handler := api.NewHandler(logger, client, manager, lifecycle, profiles, status, peers, networks, logs, api.BuildInfo{Version: version, Commit: commit, BuildTime: buildTime})
	if cfg.WebRoot != "" {
		handler = api.WithStaticFiles(handler, cfg.GatewayPrefix, cfg.WebRoot)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	listener, err := listenerFor(cfg)
	if err != nil {
		logger.Error("cannot create API listener", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("starting fnOS API", "listener", listener.Addr().String())
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-signalContext.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("API shutdown failed", "error", err)
	}
}

func listenerFor(cfg config.Config) (net.Listener, error) {
	if cfg.SocketPath == "" {
		return net.Listen("tcp", cfg.ListenAddr)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0755); err != nil {
		return nil, err
	}
	if err := os.Remove(cfg.SocketPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return net.Listen("unix", cfg.SocketPath)
}
