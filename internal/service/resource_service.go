// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ResourceServiceInterface
type ResourceServiceInterface interface {
	GetResource(ctx context.Context) *v1.Resource
}

type ResourceService struct {
	info     host.InfoInterface
	resource *v1.Resource
}

func NewResourceService() *ResourceService {
	resource := &v1.Resource{
		Id: "",
		Instances: []*v1.Instance{
			{
				InstanceMeta: &v1.InstanceMeta{
					// InstanceId:   gc.config.UUID,
					InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
					// Version:      gc.config.Version,
				},
				InstanceConfig: &v1.InstanceConfig{},
			},
		},
		// Info: host.NewInfo(),
	}

	return &ResourceService{
		info:     host.NewInfo(),
		resource: resource,
	}
}

func (rs *ResourceService) GetResource(ctx context.Context) *v1.Resource {
	resource := &v1.Resource{
		Id: "",
		Instances: []*v1.Instance{
			{
				InstanceMeta: &v1.InstanceMeta{
					// InstanceId:   gc.config.UUID,
					InstanceType: v1.InstanceMeta_INSTANCE_TYPE_AGENT,
					// Version:      gc.config.Version,
				},
				InstanceConfig: &v1.InstanceConfig{},
			},
		},
		Info: nil,
	}

	if rs.info.IsContainer() {
		resource.Info = rs.info.GetContainerInfo()
		resource.Id = resource.GetContainerInfo().GetId()
	} else {
		resource.Info = rs.info.GetHostInfo(ctx)
		resource.Id = resource.GetHostInfo().GetId()
	}

	return resource
}
