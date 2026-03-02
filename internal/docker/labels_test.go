package docker

import (
	"testing"
)

func TestParseLabels(t *testing.T) {
	labels := map[string]string{
		"wingstation.group":       "Media",
		"wingstation.name":        "Plex",
		"wingstation.icon":        "🎬",
		"wingstation.description": "Media server",
		"wingstation.url":         "http://plex.local:32400",
		"wingstation.priority":    "10",
		"wingstation.hide":        "false",
		"wingstation.tags":        "media, streaming, entertainment",
		"com.docker.compose":      "ignored",
	}

	ws := ParseLabels(labels)

	if ws.Group != "Media" {
		t.Errorf("Group = %q, want %q", ws.Group, "Media")
	}
	if ws.Name != "Plex" {
		t.Errorf("Name = %q, want %q", ws.Name, "Plex")
	}
	if ws.Icon != "🎬" {
		t.Errorf("Icon = %q, want %q", ws.Icon, "🎬")
	}
	if ws.Description != "Media server" {
		t.Errorf("Description = %q, want %q", ws.Description, "Media server")
	}
	if ws.URL != "http://plex.local:32400" {
		t.Errorf("URL = %q, want %q", ws.URL, "http://plex.local:32400")
	}
	if ws.Priority != 10 {
		t.Errorf("Priority = %d, want %d", ws.Priority, 10)
	}
	if ws.Hide != false {
		t.Errorf("Hide = %v, want false", ws.Hide)
	}
	if len(ws.Tags) != 3 {
		t.Fatalf("Tags len = %d, want 3", len(ws.Tags))
	}
	if ws.Tags[0] != "media" || ws.Tags[1] != "streaming" || ws.Tags[2] != "entertainment" {
		t.Errorf("Tags = %v, want [media streaming entertainment]", ws.Tags)
	}
}

func TestParseLabelsDefaults(t *testing.T) {
	ws := ParseLabels(nil)

	if ws.Group != "" {
		t.Errorf("Group = %q, want empty", ws.Group)
	}
	if ws.Priority != 50 {
		t.Errorf("Priority = %d, want 50", ws.Priority)
	}
	if ws.Hide != false {
		t.Errorf("Hide = %v, want false", ws.Hide)
	}
}

func TestParseLabelsHide(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}

	for _, tt := range tests {
		ws := ParseLabels(map[string]string{"wingstation.hide": tt.val})
		if ws.Hide != tt.want {
			t.Errorf("Hide(%q) = %v, want %v", tt.val, ws.Hide, tt.want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	ws := WingStationLabels{Name: "Custom Name"}
	if got := ws.DisplayName("container-name"); got != "Custom Name" {
		t.Errorf("DisplayName = %q, want %q", got, "Custom Name")
	}

	ws2 := WingStationLabels{}
	if got := ws2.DisplayName("container-name"); got != "container-name" {
		t.Errorf("DisplayName = %q, want %q", got, "container-name")
	}
}

func TestGroupName(t *testing.T) {
	ws := WingStationLabels{Group: "Media"}
	if got := ws.GroupName(); got != "Media" {
		t.Errorf("GroupName = %q, want %q", got, "Media")
	}

	ws2 := WingStationLabels{}
	if got := ws2.GroupName(); got != "Ungrouped" {
		t.Errorf("GroupName = %q, want %q", got, "Ungrouped")
	}
}

func TestHasTag(t *testing.T) {
	ws := WingStationLabels{Tags: []string{"media", "streaming"}}

	if !ws.HasTag("media") {
		t.Error("HasTag(media) = false, want true")
	}
	if !ws.HasTag("MEDIA") {
		t.Error("HasTag(MEDIA) = false, want true (case-insensitive)")
	}
	if ws.HasTag("gaming") {
		t.Error("HasTag(gaming) = true, want false")
	}
}
