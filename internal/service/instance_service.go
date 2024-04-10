// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . InstanceServiceInterface
type InstanceServiceInterface interface {
	GetInstances(ctx context.Context, processes []*model.Process) []*v1.Instance
	GetInstance(instanceID string) *v1.Instance
}

type InstanceService struct {
	instances                 []*v1.Instance
	dataPlaneInstanceServices []instance.DataPlaneInstanceService
}

func NewInstanceService() *InstanceService {
	return &InstanceService{
		instances: []*v1.Instance{},
		dataPlaneInstanceServices: []instance.DataPlaneInstanceService{
			instance.NewNginx(instance.NginxParameters{}),
		},
	}
}

func (is *InstanceService) GetInstances(ctx context.Context, processes []*model.Process) []*v1.Instance {
	newInstances := []*v1.Instance{}

	for _, dataPlaneInstanceService := range is.dataPlaneInstanceServices {
		newInstances = append(newInstances, dataPlaneInstanceService.GetInstances(ctx, processes)...)
	}

	is.instances = newInstances

	return is.instances
}

func (is *InstanceService) GetInstance(instanceID string) *v1.Instance {
	for _, instanceEntity := range is.instances {
		if instanceEntity.GetInstanceMeta().GetInstanceId() == instanceID {
			return instanceEntity
		}
	}

	return nil
}
