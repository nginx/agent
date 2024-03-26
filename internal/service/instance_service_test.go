// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
	"github.com/nginx/agent/v3/internal/service/instance/instancefakes"
	"github.com/stretchr/testify/assert"
)

var testInstances = []*v1.Instance{
	{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
	},
}

func TestInstanceService_GetInstances(t *testing.T) {
	fakeDataPlaneService := &instancefakes.FakeDataPlaneInstanceService{}
	fakeDataPlaneService.GetInstancesReturns(testInstances)

	instanceService := NewInstanceService()
	instanceService.dataPlaneInstanceServices = []instance.DataPlaneInstanceService{fakeDataPlaneService}

	assert.Equal(t, testInstances, instanceService.GetInstances([]*model.Process{}))
}

func TestInstanceService_GetInstance(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.instances = testInstances

	assert.Equal(t, testInstances[0], instanceService.GetInstance("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"))
	assert.Nil(t, instanceService.GetInstance("unknown"))
}
