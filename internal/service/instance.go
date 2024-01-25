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

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
)

const (
	tenantId = ("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_instance.go . InstanceServiceInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/service mock_instance.go | sed -e s\\/service\\\\.\\/\\/g > mock_instance_fixed.go"
//go:generate mv mock_instance_fixed.go mock_instance.go
type InstanceServiceInterface interface {
	UpdateInstances(newInstances []*instances.Instance)
	GetInstances() []*instances.Instance
	UpdateInstanceConfiguration(instanceId string, location string, cachePath string) (string, error)
}

type InstanceService struct {
	instances      []*instances.Instance
	nginxInstances map[string]*instances.Instance
}

func NewInstanceService() *InstanceService {
	return &InstanceService{
		nginxInstances: make(map[string]*instances.Instance),
	}
}

func (is *InstanceService) UpdateInstances(newInstances []*instances.Instance) {
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
	return is.instances
}

func (is *InstanceService) UpdateInstanceConfiguration(instanceId string, location string, cachePath string) (correlationId string, err error) {
	correlationId = uuid.New().String()
	if _, ok := is.nginxInstances[instanceId]; ok {
		// TODO update NGINX instance configuration
	} else {
		return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("unable to find instance with id %s", instanceId)}
	}
	return correlationId, err
}
