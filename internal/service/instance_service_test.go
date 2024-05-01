// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/instance"
	"github.com/nginx/agent/v3/internal/service/instance/instancefakes"
	"github.com/stretchr/testify/assert"
)

func TestInstanceService_GetInstances(t *testing.T) {
	ctx := context.Background()

	fakeDataPlaneService := &instancefakes.FakeDataPlaneInstanceService{}
	fakeDataPlaneService.GetInstancesReturns([]*v1.Instance{protos.GetNginxOssInstance([]string{})})

	instanceService := NewInstanceService(types.GetAgentConfig())
	instanceService.dataPlaneInstanceServices = []instance.DataPlaneInstanceService{fakeDataPlaneService}

	assert.Equal(t, []*v1.Instance{protos.GetNginxOssInstance([]string{})},
		instanceService.GetInstances(ctx, make(map[int32]*model.Process)))
}

func TestInstanceService_GetInstance(t *testing.T) {
	instanceService := NewInstanceService(types.GetAgentConfig())
	instanceService.instances = []*v1.Instance{protos.GetNginxPlusInstance([]string{})}

	assert.Equal(t, protos.GetNginxPlusInstance([]string{}),
		instanceService.GetInstance(
			protos.GetNginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId()))
	assert.Nil(t, instanceService.GetInstance("unknown"))
}
