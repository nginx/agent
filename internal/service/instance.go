/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_instance.go . InstanceServiceInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/service mock_instance.go | sed -e s\\/service\\\\.\\/\\/g > mock_instance_fixed.go"
//go:generate mv mock_instance_fixed.go mock_instance.go
type InstanceServiceInterface interface {
	UpdateInstances(newInstances []*instances.Instance)
	GetInstances() []*instances.Instance
	UpdateInstanceConfiguration(instanceId string, location string) (correlationId string, err error)
	GetInstanceConfigurationStatus(instanceId string) (status *instances.ConfigurationStatus, err error)
}

type InstanceService struct {
	instances                 []*instances.Instance
	nginxInstances            map[string]*instances.Instance
	configurationStatuses     map[string]*instances.ConfigurationStatus
	instancesLock             sync.RWMutex
	configurationStatusesLock sync.RWMutex
}

func NewInstanceService() *InstanceService {
	return &InstanceService{
		nginxInstances:            make(map[string]*instances.Instance),
		configurationStatuses:     make(map[string]*instances.ConfigurationStatus),
		instancesLock:             sync.RWMutex{},
		configurationStatusesLock: sync.RWMutex{},
	}
}

func (is *InstanceService) UpdateInstances(newInstances []*instances.Instance) {
	is.instancesLock.Lock()
	defer is.instancesLock.Unlock()

	is.instances = newInstances
	if is.instances != nil {
		for _, instance := range is.instances {
			if instance.Type == instances.Type_NGINX || instance.Type == instances.Type_NGINXPLUS {
				is.nginxInstances[instance.InstanceId] = instance
			}
		}
	}
}

func (is *InstanceService) GetInstances() []*instances.Instance {
	is.instancesLock.RLock()
	defer is.instancesLock.RUnlock()

	return is.instances
}

func (is *InstanceService) GetInstance(instanceId string) *instances.Instance {
	is.instancesLock.RLock()
	defer is.instancesLock.RUnlock()

	return is.nginxInstances[instanceId]
}

func (is *InstanceService) UpdateInstanceConfiguration(instanceId string, location string) (correlationId string, err error) {
	correlationId = uuid.New().String()

	nginxInstance := is.GetInstance(instanceId)
	if nginxInstance != nil {
		is.updateConfigurationStatus(&instances.ConfigurationStatus{
			InstanceId:    instanceId,
			CorrelationId: instanceId,
			LateUpdated:   timestamppb.Now(),
			Status:        instances.Status_IN_PROGRESS,
			Message:       "Instance configuration update in progress",
		})
		// TODO update NGINX instance configuration
	} else {
		return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("unable to find instance with id %s", instanceId)}
	}

	return correlationId, err
}

func (is *InstanceService) GetInstanceConfigurationStatus(instanceId string) (status *instances.ConfigurationStatus, err error) {
	if status := is.getConfigurationStatus(instanceId); status != nil {
		return status, nil
	} else {
		return nil, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("unable to find instance with id %s", instanceId)}
	}
}

func (is *InstanceService) getConfigurationStatus(instanceId string) *instances.ConfigurationStatus {
	is.configurationStatusesLock.RLock()
	defer is.configurationStatusesLock.RUnlock()

	if status, ok := is.configurationStatuses[instanceId]; ok {
		return status
	} else {
		return nil
	}
}

func (is *InstanceService) updateConfigurationStatus(status *instances.ConfigurationStatus) {
	is.configurationStatusesLock.Lock()
	defer is.configurationStatusesLock.Unlock()

	is.configurationStatuses[status.InstanceId] = status
}
