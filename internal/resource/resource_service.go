// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/internal/datasource/host"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . resourceServiceInterface
type resourceServiceInterface interface {
	AddInstances(instanceList []*mpi.Instance) *mpi.Resource
	UpdateInstances(instanceList []*mpi.Instance) *mpi.Resource
	DeleteInstances(instanceList []*mpi.Instance) *mpi.Resource
	Apply(ctx context.Context, instanceID string) error
}

type (
	instanceOperator interface {
		Validate(ctx context.Context, instance *mpi.Instance) error
		Reload(ctx context.Context, instance *mpi.Instance) error
	}

	logTailerOperator interface {
		Tail(ctx context.Context, errorLogs string, errorChannel chan error)
	}
)

type ResourceService struct {
	info              host.InfoInterface
	resource          *mpi.Resource
	resourceMutex     sync.Mutex
	operatorsMutex    sync.Mutex
	instanceOperators map[string]instanceOperator // key is instance ID
	logTailer         logTailerOperator
	agentConfig       *config.Config
}

func NewResourceService(ctx context.Context, agentConfig *config.Config) *ResourceService {
	resourceService := &ResourceService{
		resource:          &mpi.Resource{},
		resourceMutex:     sync.Mutex{},
		info:              host.NewInfo(),
		operatorsMutex:    sync.Mutex{},
		instanceOperators: make(map[string]instanceOperator),
		logTailer:         NewLogTailerOperator(agentConfig),
		agentConfig:       agentConfig,
	}

	resourceService.updateResourceInfo(ctx)

	return resourceService
}

func (r *ResourceService) AddInstances(instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()
	r.resource.Instances = append(r.resource.GetInstances(), instanceList...)
	r.AddOperator(instanceList)

	return r.resource
}

func (r *ResourceService) AddOperator(instanceList []*mpi.Instance) {
	r.operatorsMutex.Lock()
	defer r.operatorsMutex.Unlock()
	for _, instance := range instanceList {
		r.instanceOperators[instance.GetInstanceMeta().GetInstanceId()] = NewInstanceOperator()
	}
}

func (r *ResourceService) RemoveOperator(instanceList []*mpi.Instance) {
	r.operatorsMutex.Lock()
	defer r.operatorsMutex.Unlock()
	for _, instance := range instanceList {
		delete(r.instanceOperators, instance.GetInstanceMeta().GetInstanceId())
	}
}

func (r *ResourceService) UpdateInstances(instanceList []*mpi.Instance) *mpi.Resource {
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

func (r *ResourceService) DeleteInstances(instanceList []*mpi.Instance) *mpi.Resource {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	for _, deletedInstance := range instanceList {
		for index, instance := range r.resource.GetInstances() {
			if deletedInstance.GetInstanceMeta().GetInstanceId() == instance.GetInstanceMeta().GetInstanceId() {
				r.resource.Instances = append(r.resource.Instances[:index], r.resource.GetInstances()[index+1:]...)
			}
		}
	}
	r.RemoveOperator(instanceList)

	return r.resource
}

// nolint: revive
func (r *ResourceService) Apply(ctx context.Context, instanceID string) error {
	var instance *mpi.Instance
	var errorLogs []string
	var errorsFound error
	operator := r.instanceOperators[instanceID]
	for _, resourceInstance := range r.resource.GetInstances() {
		if resourceInstance.GetInstanceMeta().GetInstanceId() == instanceID {
			instance = resourceInstance
		}
	}

	valErr := operator.Validate(ctx, instance)
	if valErr != nil {
		return valErr
	}

	reloadErr := operator.Reload(ctx, instance)
	if reloadErr != nil {
		return reloadErr
	}

	if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		errorLogs = instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetErrorLogs()
	} else if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX {
		errorLogs = instance.GetInstanceRuntime().GetNginxRuntimeInfo().GetErrorLogs()
	}

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go r.monitorLogs(ctx, errorLogs, logErrorChannel)

	numberOfExpectedMessages := len(errorLogs)

	for i := 0; i < numberOfExpectedMessages; i++ {
		err := <-logErrorChannel
		slog.DebugContext(ctx, "Message received in logErrorChannel", "error", err)
		if err != nil {
			errorsFound = errors.Join(errorsFound, err)
		}
	}

	slog.InfoContext(ctx, "Finished monitoring post reload")

	if errorsFound != nil {
		return errorsFound
	}

	return nil
}

func (r *ResourceService) monitorLogs(ctx context.Context, errorLogs []string, errorChannel chan error) {
	if len(errorLogs) == 0 {
		slog.InfoContext(ctx, "No NGINX error logs found to monitor")
		return
	}

	for _, errorLog := range errorLogs {
		go r.logTailer.Tail(ctx, errorLog, errorChannel)
	}
}

func (r *ResourceService) updateResourceInfo(ctx context.Context) {
	r.resourceMutex.Lock()
	defer r.resourceMutex.Unlock()

	if r.info.IsContainer() {
		r.resource.Info = r.info.ContainerInfo()
		r.resource.ResourceId = r.resource.GetContainerInfo().GetContainerId()
		r.resource.Instances = []*mpi.Instance{}
	} else {
		r.resource.Info = r.info.HostInfo(ctx)
		r.resource.ResourceId = r.resource.GetHostInfo().GetHostId()
		r.resource.Instances = []*mpi.Instance{}
	}
}
