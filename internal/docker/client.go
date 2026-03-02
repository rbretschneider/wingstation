package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

// ReadOnlyClient exposes ONLY read methods from the Docker API.
// No mutation methods (start, stop, remove, etc.) are included.
type ReadOnlyClient interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
	Info(ctx context.Context) (system.Info, error)
	DiskUsage(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error)
	Events(ctx context.Context, options events.ListOptions) (<-chan events.Message, <-chan error)
	Ping(ctx context.Context) (types.Ping, error)
	Close() error
}

// Compile-time check: the Docker SDK client satisfies ReadOnlyClient.
var _ ReadOnlyClient = (*client.Client)(nil)

// NewReadOnlyClient creates a Docker client connected via the given socket path.
// The returned client only exposes read operations through the ReadOnlyClient interface.
func NewReadOnlyClient(socketPath string) (ReadOnlyClient, error) {
	c, err := client.NewClientWithOpts(
		client.WithHost("unix://"+socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// DrainBody reads and closes a ReadCloser (used for stats responses).
func DrainBody(rc io.ReadCloser) {
	if rc != nil {
		_, _ = io.Copy(io.Discard, rc)
		_ = rc.Close()
	}
}
