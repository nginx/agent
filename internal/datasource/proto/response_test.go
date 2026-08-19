// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"strings"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDataPlaneResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		correlationID  string
		instanceID     string
		requestType    mpi.DataPlaneResponse_RequestType
		commandStatus  mpi.CommandResponse_CommandStatus
		nilCmdResponse bool
	}{
		{
			name:          "Test 1: config apply request",
			correlationID: "corr-001",
			instanceID:    "inst-001",
			requestType:   mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 2: config upload request",
			correlationID: "corr-002",
			instanceID:    "inst-002",
			requestType:   mpi.DataPlaneResponse_CONFIG_UPLOAD_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 3: health request",
			correlationID: "corr-003",
			instanceID:    "inst-003",
			requestType:   mpi.DataPlaneResponse_HEALTH_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_ERROR,
		},
		{
			name:          "Test 4: status request",
			correlationID: "corr-004",
			instanceID:    "inst-004",
			requestType:   mpi.DataPlaneResponse_STATUS_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 5: api action request",
			correlationID: "corr-005",
			instanceID:    "inst-005",
			requestType:   mpi.DataPlaneResponse_API_ACTION_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 6: command status request",
			correlationID: "corr-006",
			instanceID:    "inst-006",
			requestType:   mpi.DataPlaneResponse_COMMAND_STATUS_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 7: update agent config request",
			correlationID: "corr-007",
			instanceID:    "inst-007",
			requestType:   mpi.DataPlaneResponse_UPDATE_AGENT_CONFIG_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 8: unspecified request type",
			correlationID: "corr-008",
			instanceID:    "inst-008",
			requestType:   mpi.DataPlaneResponse_UNSPECIFIED_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 9: command status unspecified",
			correlationID: "corr-009",
			instanceID:    "inst-009",
			requestType:   mpi.DataPlaneResponse_STATUS_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_UNSPECIFIED,
		},
		{
			name:          "Test 10: command status in progress",
			correlationID: "corr-010",
			instanceID:    "inst-010",
			requestType:   mpi.DataPlaneResponse_STATUS_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_IN_PROGRESS,
		},
		{
			name:          "Test 11: command status failure",
			correlationID: "corr-011",
			instanceID:    "inst-011",
			requestType:   mpi.DataPlaneResponse_STATUS_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		},
		{
			name:          "Test 12: empty correlation ID with populated instance ID",
			correlationID: "",
			instanceID:    "inst-abc",
			requestType:   mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 13: empty instance ID with populated correlation ID",
			correlationID: "corr-abc",
			instanceID:    "",
			requestType:   mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 14: both IDs empty",
			correlationID: "",
			instanceID:    "",
			requestType:   mpi.DataPlaneResponse_UNSPECIFIED_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 15: IDs with special characters",
			correlationID: "corr/id?foo=bar&baz=1 <test>",
			instanceID:    `inst\uuid:漢字/émoji`,
			requestType:   mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:          "Test 16: IDs at 256-character boundary",
			correlationID: strings.Repeat("c", 256),
			instanceID:    strings.Repeat("i", 256),
			requestType:   mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			commandStatus: mpi.CommandResponse_COMMAND_STATUS_OK,
		},
		{
			name:           "Test 17: nil command response",
			correlationID:  "corr-nil",
			instanceID:     "inst-nil",
			requestType:    mpi.DataPlaneResponse_STATUS_REQUEST,
			nilCmdResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cmdResponse *mpi.CommandResponse
			if !tt.nilCmdResponse {
				cmdResponse = &mpi.CommandResponse{Status: tt.commandStatus}
			}

			resp := CreateDataPlaneResponse(tt.correlationID, cmdResponse, tt.requestType, tt.instanceID)

			require.NotNil(t, resp)
			require.NotNil(t, resp.GetMessageMeta())
			assert.NotEmpty(t, resp.GetMessageMeta().GetMessageId())
			assert.Equal(t, tt.correlationID, resp.GetMessageMeta().GetCorrelationId())
			assert.NotNil(t, resp.GetMessageMeta().GetTimestamp())
			assert.Equal(t, tt.instanceID, resp.GetInstanceId())
			assert.Equal(t, tt.requestType, resp.GetRequestType())
			if !tt.nilCmdResponse {
				assert.Equal(t, tt.commandStatus, resp.GetCommandResponse().GetStatus())
			} else {
				assert.Nil(t, resp.GetCommandResponse())
			}
		})
	}
}

func TestCreateDataPlaneResponse_UniqueMessageIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		r1   *mpi.DataPlaneResponse
		r2   *mpi.DataPlaneResponse
		name string
	}{
		{
			name: "Test 1: distinct correlation and instance IDs",
			r1:   CreateDataPlaneResponse("c1", nil, mpi.DataPlaneResponse_STATUS_REQUEST, "i1"),
			r2:   CreateDataPlaneResponse("c2", nil, mpi.DataPlaneResponse_STATUS_REQUEST, "i2"),
		},
		{
			name: "Test 2: shared instance ID",
			r1:   CreateDataPlaneResponse("c1", nil, mpi.DataPlaneResponse_STATUS_REQUEST, "inst-shared"),
			r2:   CreateDataPlaneResponse("c2", nil, mpi.DataPlaneResponse_STATUS_REQUEST, "inst-shared"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEqual(t, tt.r1.GetMessageMeta().GetMessageId(), tt.r2.GetMessageMeta().GetMessageId())
		})
	}
}
