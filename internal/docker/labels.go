package docker

import (
	"strconv"
	"strings"
)

const labelPrefix = "wingstation."

// WingStationLabels holds parsed wingstation.* label values.
type WingStationLabels struct {
	Group       string
	Name        string
	Icon        string
	Description string
	URL         string
	Priority    int
	Hide        bool
	Tags        []string
}

// ParseLabels extracts wingstation.* labels from a label map.
func ParseLabels(labels map[string]string) WingStationLabels {
	ws := WingStationLabels{
		Priority: 50, // default mid-priority
	}

	for k, v := range labels {
		if !strings.HasPrefix(k, labelPrefix) {
			continue
		}
		key := strings.TrimPrefix(k, labelPrefix)
		switch key {
		case "group":
			ws.Group = v
		case "name":
			ws.Name = v
		case "icon":
			ws.Icon = v
		case "description":
			ws.Description = v
		case "url":
			ws.URL = v
		case "priority":
			if n, err := strconv.Atoi(v); err == nil {
				ws.Priority = n
			}
		case "hide":
			ws.Hide = v == "true" || v == "1" || v == "yes"
		case "tags":
			for _, tag := range strings.Split(v, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					ws.Tags = append(ws.Tags, tag)
				}
			}
		}
	}

	return ws
}

// DisplayName returns the best display name for a container.
func (w WingStationLabels) DisplayName(containerName string) string {
	if w.Name != "" {
		return w.Name
	}
	return containerName
}

// GroupName returns the group name or "Ungrouped" as default.
func (w WingStationLabels) GroupName() string {
	if w.Group != "" {
		return w.Group
	}
	return "Ungrouped"
}

// HasTag checks if a specific tag is present.
func (w WingStationLabels) HasTag(tag string) bool {
	for _, t := range w.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
