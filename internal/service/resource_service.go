// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"sync"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ResourceServiceInterface
type ResourceServiceInterface interface {
	GetResource(ctx context.Context) *v1.Resource
}

type ResourceService struct {
	info          host.InfoInterface
	resource      *v1.Resource
	resourceMutex sync.Mutex
}

func NewResourceService() *ResourceService {
	return &ResourceService{
		resource: &v1.Resource{
			Instances: []*v1.Instance{},
		},
		resourceMutex: sync.Mutex{},
		info:          host.NewInfo(),
	}
}

func (rs *ResourceService) GetResource(ctx context.Context) *v1.Resource {
	rs.resourceMutex.Lock()
	defer rs.resourceMutex.Unlock()

	if rs.info.IsContainer() {
		rs.resource.Info = rs.info.ContainerInfo()
		rs.resource.ResourceId = rs.resource.GetContainerInfo().GetContainerId()

		rs.resource.Instances = []*v1.Instance{}
	} else {
		rs.resource.Info = rs.info.HostInfo(ctx)
		rs.resource.ResourceId = rs.resource.GetHostInfo().GetHostId()
		rs.resource.Instances = []*v1.Instance{}
	}

	return rs.resource
}
