package docker

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/system"
)

// Container represents a summarized container for dashboard display.
type Container struct {
	ID        string
	ShortID   string
	Name      string
	Image     string
	State     string // running, exited, paused, restarting, etc.
	Status    string // human-readable status from Docker
	Created   time.Time
	Uptime    time.Duration
	Ports     []PortMapping
	Labels    WingStationLabels
	AllLabels map[string]string
}

// ContainerGroup holds containers grouped by their wingstation.group label.
type ContainerGroup struct {
	Name       string
	Icon       string
	Priority   int
	Containers []Container
}

// ContainerDetail holds full inspection data for the detail panel.
type ContainerDetail struct {
	Container

	// General
	FullID        string
	ImageID       string
	RestartPolicy string
	RestartCount  int
	Platform      string
	Driver        string
	Entrypoint    []string
	Command       []string
	WorkingDir    string
	EnvVars       []EnvVar

	// Networking
	Hostname    string
	Domainname  string
	DNSServers  []string
	DNSSearch   []string
	NetworkMode string
	Networks    []NetworkAttachment
	ExtraHosts  []string

	// Volumes
	Mounts []MountDetail

	// Security
	Privileged   bool
	ReadOnlyRoot bool
	User         string
	CapAdd       []string
	CapDrop      []string
	AppArmor     string
	Seccomp      string
	CgroupParent string
	PidMode      string
	IpcMode      string

	// Hardware
	CPUShares   int64
	CPUQuota    int64
	CPUPeriod   int64
	CPUSetCPUs  string
	MemoryLimit int64
	MemorySwap  int64
	NanoCPUs    int64
	GPUs        string
	Devices     []DeviceMapping

	// Health
	HealthCheck   string
	HealthStatus  string
	HealthRetries int
}

// PortMapping represents a port binding.
type PortMapping struct {
	ContainerPort string
	HostPort      string
	Protocol      string
	HostIP        string
}

// NetworkAttachment describes a container's connection to a Docker network.
type NetworkAttachment struct {
	Name      string
	NetworkID string
	IPAddress string
	Gateway   string
	MacAddr   string
}

// MountDetail describes a volume or bind mount.
type MountDetail struct {
	Type        string // bind, volume, tmpfs
	Source      string
	Destination string
	ReadOnly    bool
	Driver      string
	Propagation string
}

// DeviceMapping represents a host device mapped into the container.
type DeviceMapping struct {
	Host      string
	Container string
	Perms     string
}

// EnvVar holds a parsed KEY=VALUE environment variable.
type EnvVar struct {
	Key   string
	Value string
}

// HostInfo holds Docker host and daemon information.
type HostInfo struct {
	Hostname        string
	OS              string
	Architecture    string
	KernelVersion   string
	DockerVersion   string
	APIVersion      string
	TotalMemory     int64
	CPUs            int
	ContainersTotal int
	ContainersRun   int
	ContainersPause int
	ContainersStop  int
	ImagesTotal     int
	StorageDriver   string
	DockerRootDir   string
	NetworkDrivers  []string
	VolumeDrivers   []string

	// Disk usage
	ImagesSize      int64
	ContainersSize  int64
	VolumesSize     int64
	BuildCacheSize  int64
}

// ContainerFromAPI converts a Docker API container to our domain type.
func ContainerFromAPI(c types.Container) Container {
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	created := time.Unix(c.Created, 0)
	var uptime time.Duration
	if c.State == "running" {
		uptime = time.Since(created)
	}

	seen := make(map[string]struct{})
	ports := make([]PortMapping, 0, len(c.Ports))
	for _, p := range c.Ports {
		if p.PublicPort == 0 {
			continue // skip unexposed container-only ports
		}
		key := fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		ports = append(ports, PortMapping{
			ContainerPort: portStr(p.PrivatePort, p.Type),
			HostPort:      portStr(p.PublicPort, ""),
			Protocol:      p.Type,
			HostIP:        p.IP,
		})
	}

	return Container{
		ID:        c.ID,
		ShortID:   c.ID[:12],
		Name:      name,
		Image:     c.Image,
		State:     c.State,
		Status:    c.Status,
		Created:   created,
		Uptime:    uptime,
		Ports:     ports,
		Labels:    ParseLabels(c.Labels),
		AllLabels: c.Labels,
	}
}

// HostInfoFromAPI converts Docker system info to our domain type.
func HostInfoFromAPI(info system.Info, ver string) HostInfo {
	return HostInfo{
		Hostname:        info.Name,
		OS:              info.OperatingSystem,
		Architecture:    info.Architecture,
		KernelVersion:   info.KernelVersion,
		DockerVersion:   info.ServerVersion,
		APIVersion:      ver,
		TotalMemory:     info.MemTotal,
		CPUs:            info.NCPU,
		ContainersTotal: info.Containers,
		ContainersRun:   info.ContainersRunning,
		ContainersPause: info.ContainersPaused,
		ContainersStop:  info.ContainersStopped,
		ImagesTotal:     info.Images,
		StorageDriver:   info.Driver,
		DockerRootDir:   info.DockerRootDir,
	}
}

func portStr(port uint16, proto string) string {
	if port == 0 {
		return ""
	}
	if proto != "" {
		return fmt.Sprintf("%d/%s", port, proto)
	}
	return fmt.Sprintf("%d", port)
}
