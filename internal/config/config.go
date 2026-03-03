package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port    int
	Host    string
	BaseURL string

	// Docker
	DockerSocket string

	// Auth
	AuthEnabled bool
	AuthUser    string
	AuthPass    string

	// Cache
	CacheTTL time.Duration

	// SSE
	SSEEnabled      bool
	SSERetryMs      int
	StatsIntervalMs int

	// Terminal
	TerminalEnabled bool

	// Logging
	LogLevel string
}

// Load reads configuration from WINGSTATION_* environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:            envInt("WINGSTATION_PORT", 8080),
		Host:            envStr("WINGSTATION_HOST", "0.0.0.0"),
		BaseURL:         envStr("WINGSTATION_BASE_URL", ""),
		DockerSocket:    envStr("WINGSTATION_DOCKER_SOCKET", "/var/run/docker.sock"),
		AuthEnabled:     envBool("WINGSTATION_AUTH_ENABLED", false),
		AuthUser:        envStr("WINGSTATION_AUTH_USER", ""),
		AuthPass:        envStr("WINGSTATION_AUTH_PASS", ""),
		CacheTTL:        envDuration("WINGSTATION_CACHE_TTL", 5*time.Second),
		SSEEnabled:      envBool("WINGSTATION_SSE_ENABLED", true),
		SSERetryMs:      envInt("WINGSTATION_SSE_RETRY_MS", 3000),
		StatsIntervalMs: envInt("WINGSTATION_STATS_INTERVAL_MS", 5000),
		TerminalEnabled: envBool("WINGSTATION_TERMINAL_ENABLED", false),
		LogLevel:        envStr("WINGSTATION_LOG_LEVEL", "info"),
	}
}

// Addr returns the listen address in host:port format.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
