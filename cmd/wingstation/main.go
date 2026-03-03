package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rbretschneider/wingstation/internal/cache"
	"github.com/rbretschneider/wingstation/internal/config"
	"github.com/rbretschneider/wingstation/internal/docker"
	"github.com/rbretschneider/wingstation/internal/server"
	"github.com/rbretschneider/wingstation/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Configure logging
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	slog.Info("Starting WingStation",
		"port", cfg.Port,
		"docker_socket", cfg.DockerSocket,
		"sse_enabled", cfg.SSEEnabled,
		"terminal_enabled", cfg.TerminalEnabled,
	)

	// Create Docker client
	dockerClient, err := docker.NewReadOnlyClient(cfg.DockerSocket)
	if err != nil {
		slog.Error("Failed to create Docker client", "error", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Verify Docker connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = dockerClient.Ping(ctx)
	cancel()
	if err != nil {
		slog.Error("Failed to connect to Docker daemon", "error", err, "socket", cfg.DockerSocket)
		os.Exit(1)
	}
	slog.Info("Connected to Docker daemon")

	// Create exec client for terminal (optional)
	var execClient docker.ExecClient
	if cfg.TerminalEnabled {
		slog.Warn("Terminal mode is ENABLED — this breaks the read-only guarantee")
		execClient, err = docker.NewExecClient(cfg.DockerSocket)
		if err != nil {
			slog.Error("Failed to create exec client", "error", err)
			os.Exit(1)
		}
	}

	// Create cache
	appCache := cache.New(cfg.CacheTTL)
	defer appCache.Stop()

	// Create services
	containerSvc := service.NewContainerService(dockerClient, appCache)
	hostSvc := service.NewHostService(dockerClient, appCache)

	// Create and start server
	srv, err := server.New(cfg, containerSvc, hostSvc, dockerClient, execClient)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("Shutting down WingStation...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
	}

	slog.Info("WingStation stopped")
}
