package service

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/rbretschneider/wingstation/internal/cache"
	"github.com/rbretschneider/wingstation/internal/docker"
)

const cacheKeyHostInfo = "hostinfo"

// HostService provides Docker host/daemon information.
type HostService struct {
	client docker.ReadOnlyClient
	cache  *cache.Cache
}

// NewHostService creates a new host service.
func NewHostService(client docker.ReadOnlyClient, cache *cache.Cache) *HostService {
	return &HostService{client: client, cache: cache}
}

// GetInfo returns host and daemon information.
func (s *HostService) GetInfo(ctx context.Context) (*docker.HostInfo, error) {
	if cached, ok := s.cache.Get(cacheKeyHostInfo); ok {
		return cached.(*docker.HostInfo), nil
	}

	info, err := s.client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting docker info: %w", err)
	}

	ping, err := s.client.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("pinging docker: %w", err)
	}

	hostInfo := docker.HostInfoFromAPI(info, ping.APIVersion)

	// Get disk usage
	du, err := s.client.DiskUsage(ctx, types.DiskUsageOptions{})
	if err == nil {
		for _, img := range du.Images {
			hostInfo.ImagesSize += img.Size
		}
		for _, c := range du.Containers {
			hostInfo.ContainersSize += c.SizeRw
		}
		for _, v := range du.Volumes {
			if v.UsageData.Size > 0 {
				hostInfo.VolumesSize += v.UsageData.Size
			}
		}
		if du.BuildCache != nil {
			for _, bc := range du.BuildCache {
				hostInfo.BuildCacheSize += bc.Size
			}
		}
	}

	s.cache.Set(cacheKeyHostInfo, &hostInfo)
	return &hostInfo, nil
}
