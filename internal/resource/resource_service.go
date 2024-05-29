// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"sync"

	"github.com/nginx/agent/v3/internal/datasource/host"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . resourceServiceInterface
type resourceServiceInterface interface {
	AddInstances(instanceList []*v1.Instance) *v1.Resource
	UpdateInstances(instanceList []*v1.Instance) *v1.Resource
	DeleteInstances(instanceList []*v1.Instance) *v1.Resource
}

type ResourceService struct {
	info          host.InfoInterface
	resource      *v1.Resource
	resourceMutex sync.Mutex
}

func NewResourceService(ctx context.Context) *ResourceService {
	resourceService := &ResourceService{
		resource:      &v1.Resource{},
		resourceMutex: sync.Mutex{},
		info:          host.NewInfo(),
	}

	resourceService.updateResourceInfo(ctx)

	return resourceService
}

func (r *ResourceService) AddInstances(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()
	r.resource.Instances = append(r.resource.GetInstances(), instanceList...)

	return r.resource
}

func (r *ResourceService) UpdateInstances(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, updatedInstance := range instanceList {
		for _, instance := range r.resource.GetInstances() {
			if updatedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				instance.InstanceMeta = updatedInstance.GetInstanceMeta()
				instance.InstanceRuntime = updatedInstance.GetInstanceRuntime()
				instance.InstanceConfig = updatedInstance.GetInstanceConfig()
			}
		}
	}

	return r.resource
}

func (r *ResourceService) DeleteInstances(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, deletedInstance := range instanceList {
		for index, instance := range r.resource.GetInstances() {
			if deletedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				r.resource.Instances = append(r.resource.Instances[:index], r.resource.GetInstances()[index+1:]...)
			}
		}
	}

	return r.resource
}

func (r *ResourceService) updateResourceInfo(ctx context.Context) {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	if r.info.IsContainer() {
		r.resource.Info = r.info.ContainerInfo()
		r.resource.ResourceId = r.resource.GetContainerInfo().GetContainerId()
		r.resource.Instances = []*v1.Instance{}
	} else {
		r.resource.Info = r.info.HostInfo(ctx)
		r.resource.ResourceId = r.resource.GetHostInfo().GetHostId()
		r.resource.Instances = []*v1.Instance{}
	}
}
