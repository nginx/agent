// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . InstanceServiceInterface
type InstanceServiceInterface interface {
	GetInstances(processes []*model.Process) []*instances.Instance
	GetInstance(instanceID string) *instances.Instance
}

type InstanceService struct {
	instances                 []*instances.Instance
	dataplaneInstanceServices []instance.DataplaneInstanceService
}

func NewInstanceService() *InstanceService {
	return &InstanceService{
		instances: []*instances.Instance{},
		dataplaneInstanceServices: []instance.DataplaneInstanceService{
			instance.NewNginx(instance.NginxParameters{}),
		},
	}
}

func (is *InstanceService) GetInstances(processes []*model.Process) []*instances.Instance {
	newInstances := []*instances.Instance{}

	for _, dataplaneInstanceService := range is.dataplaneInstanceServices {
		newInstances = append(newInstances, dataplaneInstanceService.GetInstances(processes)...)
	}

	is.instances = newInstances

	return is.instances
}

func (is *InstanceService) GetInstance(instanceID string) *instances.Instance {
	for _, instanceEntity := range is.instances {
		if instanceEntity.GetInstanceId() == instanceID {
			return instanceEntity
		}
	}

	return nil
}
