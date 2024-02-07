// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
	"github.com/nginx/agent/v3/internal/service/instance/instancefakes"
	"github.com/stretchr/testify/assert"
)

var testInstances = []*instances.Instance{
	{
		InstanceId: "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		Type:       instances.Type_NGINX,
	},
}

func TestInstanceService_GetInstances(t *testing.T) {
	fakeDataplaneService := &instancefakes.FakeDataplaneInstanceService{}
	fakeDataplaneService.GetInstancesReturns(testInstances)

	instanceService := NewInstanceService()
	instanceService.dataplaneInstanceServices = []instance.DataplaneInstanceService{fakeDataplaneService}

	assert.Equal(t, testInstances, instanceService.GetInstances([]*model.Process{}))
}

func TestInstanceService_GetInstance(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.instances = testInstances

	assert.Equal(t, testInstances[0], instanceService.GetInstance("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"))
	assert.Nil(t, instanceService.GetInstance("unknown"))
}
