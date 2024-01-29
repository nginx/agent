/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"
)

var testInstances = []*instances.Instance{
	{
		InstanceId: "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		Type:       instances.Type_NGINX,
	},
}

func TestInstanceService_UpdateInstances(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.UpdateInstances(testInstances)
	assert.Equal(t, testInstances, instanceService.instances)
}

func TestInstanceService_GetInstances(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.UpdateInstances(testInstances)
	assert.Equal(t, testInstances, instanceService.GetInstances())
}
