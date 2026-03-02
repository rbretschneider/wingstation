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

// InferRepoURL tries to find a GitHub/source repository URL from container labels and image name.
// Priority: OCI label > image-based heuristic.
func InferRepoURL(labels map[string]string, image string) string {
	// 1. Check OCI standard labels (many images set these at build time)
	for _, key := range []string{
		"org.opencontainers.image.source",
		"org.opencontainers.image.url",
		"org.label-schema.vcs-url",
	} {
		if v, ok := labels[key]; ok && v != "" {
			return v
		}
	}

	// 2. Heuristic: derive GitHub URL from image name
	// Strip tag/digest
	img := image
	if at := strings.Index(img, "@"); at != -1 {
		img = img[:at]
	}
	if colon := strings.LastIndex(img, ":"); colon != -1 {
		img = img[:colon]
	}
	// Strip sha256: prefix if present
	img = strings.TrimPrefix(img, "sha256:")

	// Skip raw sha256 digests — no repo info
	if len(img) == 64 && !strings.Contains(img, "/") {
		return ""
	}

	// ghcr.io/owner/repo → github.com/owner/repo
	if strings.HasPrefix(img, "ghcr.io/") {
		parts := strings.SplitN(strings.TrimPrefix(img, "ghcr.io/"), "/", 3)
		if len(parts) >= 2 {
			return "https://github.com/" + parts[0] + "/" + parts[1]
		}
	}

	// lscr.io/linuxserver/name or linuxserver/name → github.com/linuxserver/docker-name
	cleaned := strings.TrimPrefix(img, "lscr.io/")
	if strings.HasPrefix(cleaned, "linuxserver/") {
		name := strings.TrimPrefix(cleaned, "linuxserver/")
		return "https://github.com/linuxserver/docker-" + name
	}

	// Docker Hub library images: nginx, alpine, etc. → github.com/docker-library/name
	if !strings.Contains(img, "/") && !strings.Contains(img, ".") {
		return "https://hub.docker.com/_/" + img
	}

	// Docker Hub user/repo → hub.docker.com/r/user/repo (with GitHub link attempt)
	if !strings.Contains(img, ".") {
		parts := strings.SplitN(img, "/", 2)
		if len(parts) == 2 {
			return "https://hub.docker.com/r/" + parts[0] + "/" + parts[1]
		}
	}

	return ""
}
