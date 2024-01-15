/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/model/os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_instance.go . InstanceServiceInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/service mock_instance.go | sed -e s\\/service\\\\.\\/\\/g > mock_instance_fixed.go"
//go:generate mv mock_instance_fixed.go mock_instance.go
type InstanceServiceInterface interface {
	UpdateProcesses(newProcesses []*os.Process)
	GetInstances() ([]*instances.Instance, error)
}

type InstanceService struct {
	processes []*os.Process
}

func NewInstanceService() *InstanceService {
	return &InstanceService{}
}

func (is *InstanceService) UpdateProcesses(newProcesses []*os.Process) {
	is.processes = newProcesses
}

func (is *InstanceService) GetInstances() ([]*instances.Instance, error) {
	n := nginx.New(nginx.NginxParameters{})
	instances, err := n.GetInstances(is.processes)
	return instances, err
}
