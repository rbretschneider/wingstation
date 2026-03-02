package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Host)
	}
	if cfg.DockerSocket != "/var/run/docker.sock" {
		t.Errorf("DockerSocket = %q, want /var/run/docker.sock", cfg.DockerSocket)
	}
	if cfg.AuthEnabled {
		t.Error("AuthEnabled should be false by default")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if !cfg.SSEEnabled {
		t.Error("SSEEnabled should be true by default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("WINGSTATION_PORT", "9090")
	os.Setenv("WINGSTATION_AUTH_ENABLED", "true")
	os.Setenv("WINGSTATION_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("WINGSTATION_PORT")
		os.Unsetenv("WINGSTATION_AUTH_ENABLED")
		os.Unsetenv("WINGSTATION_LOG_LEVEL")
	}()

	cfg := Load()

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if !cfg.AuthEnabled {
		t.Error("AuthEnabled should be true")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
}

func TestAddr(t *testing.T) {
	cfg := &Config{Host: "0.0.0.0", Port: 8080}
	if got := cfg.Addr(); got != "0.0.0.0:8080" {
		t.Errorf("Addr = %q, want 0.0.0.0:8080", got)
	}
}
