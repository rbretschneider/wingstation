package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ExecClient exposes Docker exec methods — separate from ReadOnlyClient
// because exec is a write/interactive operation that breaks the read-only guarantee.
// Only used when WINGSTATION_TERMINAL_ENABLED=true.
type ExecClient interface {
	ContainerExecCreate(ctx context.Context, ctr string, config container.ExecOptions) (types.IDResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error)
	ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
}

// Compile-time check: the Docker SDK client satisfies ExecClient.
var _ ExecClient = (*client.Client)(nil)

// NewExecClient creates a Docker client for exec operations.
func NewExecClient(socketPath string) (ExecClient, error) {
	c, err := client.NewClientWithOpts(
		client.WithHost("unix://"+socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}
