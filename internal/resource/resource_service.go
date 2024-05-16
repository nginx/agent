// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"sync"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . resourceServiceInterface
type resourceServiceInterface interface {
	AddInstance(instanceList []*v1.Instance) *v1.Resource
	UpdateInstance(instanceList []*v1.Instance) *v1.Resource
	DeleteInstance(instanceList []*v1.Instance) *v1.Resource
}

type ResourceService struct {
	resource      *v1.Resource
	resourceMutex sync.Mutex
}

func NewResourceService() *ResourceService {
	return &ResourceService{
		resource: &v1.Resource{
			Instances: []*v1.Instance{},
		},
		resourceMutex: sync.Mutex{},
	}
}

func (r *ResourceService) AddInstance(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()
	r.resource.Instances = append(r.resource.GetInstances(), instanceList...)
	r.resourceMutex.Unlock()

	return r.resource
}

func (r *ResourceService) UpdateInstance(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()

	for _, updatedInstance := range instanceList {
		for _, instance := range r.resource.GetInstances() {
			if updatedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				instance.InstanceMeta = updatedInstance.GetInstanceMeta()
				instance.InstanceRuntime = updatedInstance.GetInstanceRuntime()
				instance.InstanceConfig = updatedInstance.GetInstanceConfig()
			}
		}
	}
	r.resourceMutex.Unlock()

	return r.resource
}

func (r *ResourceService) DeleteInstance(instanceList []*v1.Instance) *v1.Resource {
	r.resourceMutex.Lock()

	for _, deletedInstance := range instanceList {
		for index, instance := range r.resource.GetInstances() {
			if deletedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				r.resource.Instances = append(r.resource.Instances[:index], r.resource.GetInstances()[index+1:]...)
			}
		}
	}

	r.resourceMutex.Unlock()

	return r.resource
}
