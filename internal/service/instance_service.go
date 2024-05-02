// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"github.com/nginx/agent/v3/internal/datasource/host"
	"sync"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/service/instance"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . InstanceServiceInterface
type InstanceServiceInterface interface {
	GetInstances(ctx context.Context, processes host.NginxProcesses) []*v1.Instance
	GetInstance(instanceID string) *v1.Instance
}

type InstanceService struct {
	instances                 []*v1.Instance
	dataPlaneInstanceServices []instance.DataPlaneInstanceService
	instancesMutex            sync.Mutex
}

func NewInstanceService(agentConfig *config.Config) *InstanceService {
	return &InstanceService{
		instances: []*v1.Instance{},
		dataPlaneInstanceServices: []instance.DataPlaneInstanceService{
			instance.NewNginxAgent(agentConfig),
			instance.NewNginx(instance.NginxParameters{}),
		},
		instancesMutex: sync.Mutex{},
	}
}

func (is *InstanceService) GetInstances(ctx context.Context, processes host.NginxProcesses) []*v1.Instance {
	newInstances := []*v1.Instance{}

	for _, dataPlaneInstanceService := range is.dataPlaneInstanceServices {
		newInstances = append(newInstances, dataPlaneInstanceService.GetInstances(ctx, processes)...)
	}

	is.instancesMutex.Lock()
	is.instances = newInstances
	is.instancesMutex.Unlock()

	return is.instances
}

func (is *InstanceService) GetInstance(instanceID string) *v1.Instance {
	is.instancesMutex.Lock()
	defer is.instancesMutex.Unlock()

	for _, instanceEntity := range is.instances {
		if instanceEntity.GetInstanceMeta().GetInstanceId() == instanceID {
			return instanceEntity
		}
	}

	return nil
}
