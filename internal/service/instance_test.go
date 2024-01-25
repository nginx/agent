/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/types/known/timestamppb"
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

func createTestIds() (uuid.UUID, uuid.UUID, error) {
	tenantId, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	if err != nil {
		fmt.Printf("Error creating tenantId: %v", err)
	}

	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	if err != nil {
		fmt.Printf("Error creating instanceId: %v", err)
	}

	return tenantId, instanceId, err
}

func createProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	time, err := time.Parse(time.RFC3339, timeString)
	protoTime := timestamppb.New(time)

	return protoTime, err
}
