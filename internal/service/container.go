package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rbretschneider/wingstation/internal/cache"
	"github.com/rbretschneider/wingstation/internal/docker"
)

const cacheKeyContainers = "containers"

// ContainerService provides read-only container operations.
type ContainerService struct {
	client docker.ReadOnlyClient
	cache  *cache.Cache
}

// NewContainerService creates a new container service.
func NewContainerService(client docker.ReadOnlyClient, cache *cache.Cache) *ContainerService {
	return &ContainerService{client: client, cache: cache}
}

// List returns all containers (including stopped).
func (s *ContainerService) List(ctx context.Context) ([]docker.Container, error) {
	if cached, ok := s.cache.Get(cacheKeyContainers); ok {
		return cached.([]docker.Container), nil
	}

	apiContainers, err := s.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	containers := make([]docker.Container, 0, len(apiContainers))
	for _, c := range apiContainers {
		dc := docker.ContainerFromAPI(c)
		if !dc.Labels.Hide {
			containers = append(containers, dc)
		}
	}

	s.cache.Set(cacheKeyContainers, containers)
	return containers, nil
}

// ListGrouped returns containers grouped by wingstation.group label, sorted by priority.
func (s *ContainerService) ListGrouped(ctx context.Context) ([]docker.ContainerGroup, error) {
	containers, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string]*docker.ContainerGroup)
	for _, c := range containers {
		gName := c.Labels.GroupName()
		g, ok := groupMap[gName]
		if !ok {
			g = &docker.ContainerGroup{
				Name:     gName,
				Icon:     c.Labels.Icon,
				Priority: c.Labels.Priority,
			}
			groupMap[gName] = g
		}
		g.Containers = append(g.Containers, c)
		// Use lowest priority number (highest priority) for the group
		if c.Labels.Priority < g.Priority {
			g.Priority = c.Labels.Priority
		}
	}

	groups := make([]docker.ContainerGroup, 0, len(groupMap))
	for _, g := range groupMap {
		// Sort containers within group by priority then name
		sort.Slice(g.Containers, func(i, j int) bool {
			if g.Containers[i].Labels.Priority != g.Containers[j].Labels.Priority {
				return g.Containers[i].Labels.Priority < g.Containers[j].Labels.Priority
			}
			return g.Containers[i].Name < g.Containers[j].Name
		})
		groups = append(groups, *g)
	}

	// Sort groups: by priority, then alphabetically
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Priority != groups[j].Priority {
			return groups[i].Priority < groups[j].Priority
		}
		return groups[i].Name < groups[j].Name
	})

	return groups, nil
}

// SearchParams defines filtering criteria.
type SearchParams struct {
	Query  string
	Status string
	Group  string
	Tag    string
	Sort   string
}

// Search filters containers by the given parameters.
func (s *ContainerService) Search(ctx context.Context, params SearchParams) ([]docker.Container, error) {
	containers, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []docker.Container
	for _, c := range containers {
		if !matchesSearch(c, params) {
			continue
		}
		filtered = append(filtered, c)
	}

	sortContainers(filtered, params.Sort)
	return filtered, nil
}

// Inspect returns full detail for a single container.
func (s *ContainerService) Inspect(ctx context.Context, id string) (*docker.ContainerDetail, error) {
	cacheKey := "inspect:" + id
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*docker.ContainerDetail), nil
	}

	raw, err := s.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("inspecting container %s: %w", id, err)
	}

	detail := containerDetailFromInspect(raw)
	s.cache.Set(cacheKey, detail)
	return detail, nil
}

// InvalidateCache clears the container cache.
func (s *ContainerService) InvalidateCache() {
	s.cache.InvalidateAll()
}

func matchesSearch(c docker.Container, p SearchParams) bool {
	if p.Status != "" && !strings.EqualFold(c.State, p.Status) {
		return false
	}
	if p.Group != "" && !strings.EqualFold(c.Labels.GroupName(), p.Group) {
		return false
	}
	if p.Tag != "" && !c.Labels.HasTag(p.Tag) {
		return false
	}
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		name := strings.ToLower(c.Labels.DisplayName(c.Name))
		image := strings.ToLower(c.Image)
		if !strings.Contains(name, q) && !strings.Contains(image, q) && !strings.Contains(c.ShortID, q) {
			return false
		}
	}
	return true
}

func sortContainers(containers []docker.Container, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(containers, func(i, j int) bool {
			return containers[i].Name < containers[j].Name
		})
	case "status":
		sort.Slice(containers, func(i, j int) bool {
			return containers[i].State < containers[j].State
		})
	case "created":
		sort.Slice(containers, func(i, j int) bool {
			return containers[i].Created.After(containers[j].Created)
		})
	default: // priority
		sort.Slice(containers, func(i, j int) bool {
			if containers[i].Labels.Priority != containers[j].Labels.Priority {
				return containers[i].Labels.Priority < containers[j].Labels.Priority
			}
			return containers[i].Name < containers[j].Name
		})
	}
}

func containerDetailFromInspect(raw types.ContainerJSON) *docker.ContainerDetail {
	d := &docker.ContainerDetail{}
	d.FullID = raw.ID
	d.ShortID = raw.ID[:12]
	d.Name = strings.TrimPrefix(raw.Name, "/")
	d.Image = raw.Config.Image
	d.ImageID = raw.Image
	d.State = raw.State.Status
	d.Status = raw.State.Status

	if raw.State.StartedAt != "" {
		// Parse and set created/uptime if possible
	}

	d.AllLabels = raw.Config.Labels
	d.Labels = docker.ParseLabels(raw.Config.Labels)

	// General
	if raw.HostConfig != nil && raw.HostConfig.RestartPolicy.Name != "" {
		d.RestartPolicy = string(raw.HostConfig.RestartPolicy.Name)
	}
	d.RestartCount = raw.RestartCount
	d.Platform = raw.Platform
	d.Driver = raw.Driver

	if raw.Config.Entrypoint != nil {
		d.Entrypoint = raw.Config.Entrypoint
	}
	if raw.Config.Cmd != nil {
		d.Command = raw.Config.Cmd
	}
	d.WorkingDir = raw.Config.WorkingDir

	// Parse env vars
	for _, e := range raw.Config.Env {
		parts := strings.SplitN(e, "=", 2)
		ev := docker.EnvVar{Key: parts[0]}
		if len(parts) > 1 {
			ev.Value = parts[1]
		}
		d.EnvVars = append(d.EnvVars, ev)
	}

	// Networking
	if raw.Config != nil {
		d.Hostname = raw.Config.Hostname
		d.Domainname = raw.Config.Domainname
	}
	if raw.HostConfig != nil {
		d.DNSServers = raw.HostConfig.DNS
		d.DNSSearch = raw.HostConfig.DNSSearch
		d.NetworkMode = string(raw.HostConfig.NetworkMode)
		d.ExtraHosts = raw.HostConfig.ExtraHosts
	}

	if raw.NetworkSettings != nil {
		for name, net := range raw.NetworkSettings.Networks {
			d.Networks = append(d.Networks, docker.NetworkAttachment{
				Name:      name,
				NetworkID: net.NetworkID,
				IPAddress: net.IPAddress,
				Gateway:   net.Gateway,
				MacAddr:   net.MacAddress,
			})
		}

		// Parse port mappings from inspection data
		for port, bindings := range raw.NetworkSettings.Ports {
			for _, b := range bindings {
				d.Ports = append(d.Ports, docker.PortMapping{
					ContainerPort: string(port),
					HostPort:      b.HostPort,
					Protocol:      port.Proto(),
					HostIP:        b.HostIP,
				})
			}
			if len(bindings) == 0 {
				d.Ports = append(d.Ports, docker.PortMapping{
					ContainerPort: string(port),
					Protocol:      port.Proto(),
				})
			}
		}
	}

	// Volumes / Mounts
	for _, m := range raw.Mounts {
		d.Mounts = append(d.Mounts, docker.MountDetail{
			Type:        string(m.Type),
			Source:      m.Source,
			Destination: m.Destination,
			ReadOnly:    !m.RW,
			Driver:      m.Driver,
			Propagation: string(m.Propagation),
		})
	}

	// Security
	if raw.HostConfig != nil {
		d.Privileged = raw.HostConfig.Privileged
		d.ReadOnlyRoot = raw.HostConfig.ReadonlyRootfs
		d.CapAdd = raw.HostConfig.CapAdd
		d.CapDrop = raw.HostConfig.CapDrop
		d.CgroupParent = raw.HostConfig.CgroupParent
		d.PidMode = string(raw.HostConfig.PidMode)
		d.IpcMode = string(raw.HostConfig.IpcMode)

		if raw.HostConfig.SecurityOpt != nil {
			for _, opt := range raw.HostConfig.SecurityOpt {
				if strings.HasPrefix(opt, "apparmor=") {
					d.AppArmor = strings.TrimPrefix(opt, "apparmor=")
				}
				if strings.HasPrefix(opt, "seccomp=") {
					d.Seccomp = strings.TrimPrefix(opt, "seccomp=")
				}
			}
		}
	}
	d.User = raw.Config.User

	// Hardware
	if raw.HostConfig != nil {
		d.CPUShares = raw.HostConfig.CPUShares
		d.CPUQuota = raw.HostConfig.CPUQuota
		d.CPUPeriod = raw.HostConfig.CPUPeriod
		d.CPUSetCPUs = raw.HostConfig.CpusetCpus
		d.MemoryLimit = raw.HostConfig.Memory
		d.MemorySwap = raw.HostConfig.MemorySwap
		d.NanoCPUs = raw.HostConfig.NanoCPUs

		for _, dev := range raw.HostConfig.Devices {
			d.Devices = append(d.Devices, docker.DeviceMapping{
				Host:      dev.PathOnHost,
				Container: dev.PathInContainer,
				Perms:     dev.CgroupPermissions,
			})
		}

		// GPU detection
		if raw.HostConfig.DeviceRequests != nil {
			for _, req := range raw.HostConfig.DeviceRequests {
				if req.Driver == "nvidia" || containsGPU(req.Capabilities) {
					d.GPUs = fmt.Sprintf("count=%d", req.Count)
					break
				}
			}
		}
	}

	// Health check
	if raw.Config.Healthcheck != nil {
		d.HealthCheck = strings.Join(raw.Config.Healthcheck.Test, " ")
		d.HealthRetries = raw.Config.Healthcheck.Retries
	}
	if raw.State != nil && raw.State.Health != nil {
		d.HealthStatus = raw.State.Health.Status
	}

	return d
}

func containsGPU(caps [][]string) bool {
	for _, group := range caps {
		for _, cap := range group {
			if cap == "gpu" {
				return true
			}
		}
	}
	return false
}
