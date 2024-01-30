/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"
	"log/slog"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . InstanceServiceInterface
type InstanceServiceInterface interface {
	GetInstances(processes []*model.Process) []*instances.Instance
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
		newDataplaneInstances, err := dataplaneInstanceService.GetInstances(processes)
		if err != nil {
			slog.Warn("Unable to get all instances", "dataplane type", fmt.Sprintf("%T", dataplaneInstanceService), "error", err)
		} else {
			newInstances = append(newInstances, newDataplaneInstances...)
		}
	}

	is.instances = newInstances
	return is.instances
}
