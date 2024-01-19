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
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testInstances = []*instances.Instance{
	{
		InstanceId: "123",
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

func TestInstanceService_GetInstanceConfigurationStatus(t *testing.T) {
	instanceService := NewInstanceService()

	// Test instance is not found
	result, err := instanceService.GetInstanceConfigurationStatus("123")
	assert.Nil(t, result)
	assert.Equal(t, 404, err.(*common.RequestError).StatusCode)

	// Test instance is found
	status := &instances.ConfigurationStatus{
		InstanceId:    "123",
		CorrelationId: "456",
		LateUpdated:   &timestamppb.Timestamp{},
		Status:        instances.Status_SUCCESS,
		Message:       "Success",
	}
	instanceService.configurationStatuses["123"] = status

	result, err = instanceService.GetInstanceConfigurationStatus("123")
	assert.Equal(t, status, result)
	assert.NoError(t, err)
}
