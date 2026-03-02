package server

import (
	"testing"
	"time"
)

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h 30m"},
		{25 * time.Hour, "1d 1h"},
		{48*time.Hour + 30*time.Minute, "2d 0h"},
	}

	for _, tt := range tests {
		got := formatUptime(tt.d)
		if got != tt.want {
			t.Errorf("formatUptime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		b    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024 * 1024, "1.5 GB"},
		{2 * 1024 * 1024 * 1024 * 1024, "2.0 TB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.b)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestStatusColor(t *testing.T) {
	if got := statusColor("running"); got != "bg-green-500" {
		t.Errorf("statusColor(running) = %q", got)
	}
	if got := statusColor("exited"); got != "bg-red-500" {
		t.Errorf("statusColor(exited) = %q", got)
	}
	if got := statusColor("paused"); got != "bg-yellow-500" {
		t.Errorf("statusColor(paused) = %q", got)
	}
}

func TestIsSensitiveKey(t *testing.T) {
	if !isSensitiveKey("DB_PASSWORD") {
		t.Error("DB_PASSWORD should be sensitive")
	}
	if !isSensitiveKey("API_KEY") {
		t.Error("API_KEY should be sensitive")
	}
	if isSensitiveKey("HOSTNAME") {
		t.Error("HOSTNAME should not be sensitive")
	}
}

func TestMaskValue(t *testing.T) {
	if got := maskValue("DB_PASSWORD", "supersecret"); got == "supersecret" {
		t.Error("sensitive value should be masked")
	}
	if got := maskValue("HOSTNAME", "myhost"); got != "myhost" {
		t.Errorf("non-sensitive value should not be masked, got %q", got)
	}
	if got := maskValue("TOKEN", "ab"); got != "****" {
		t.Errorf("short sensitive value should be fully masked, got %q", got)
	}
}

func TestIsSensitivePath(t *testing.T) {
	if !isSensitivePath("/var/run/docker.sock") {
		t.Error("/var/run/docker.sock should be sensitive")
	}
	if isSensitivePath("/app/data") {
		t.Error("/app/data should not be sensitive")
	}
}

func TestPercentage(t *testing.T) {
	if got := percentage(50, 100); got != 50.0 {
		t.Errorf("percentage(50, 100) = %f, want 50.0", got)
	}
	if got := percentage(1, 0); got != 0.0 {
		t.Errorf("percentage(1, 0) = %f, want 0.0", got)
	}
}
