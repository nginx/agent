// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configApplyErrorMessage = "failed to parse config invalid " +
		"number of arguments in \"worker_processes\" directive in /etc/nginx/nginx.conf:1"
)

func TestGrpc_ConfigApply(t *testing.T) {
	ctx := context.Background()
	teardownTest := utils.SetupConnectionTest(t, false, false)
	defer teardownTest(t)

	nginxInstanceID := utils.VerifyConnection(t, 2)

	responses := utils.GetManagementPlaneResponses(t, 1)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	t.Run("Test 1: No config changes", func(t *testing.T) {
		utils.ClearManagementPlaneResponses(t)
		utils.PerformConfigApply(t, nginxInstanceID)
		responses = utils.GetManagementPlaneResponses(t, 1)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply successful, no files to change", responses[0].GetCommandResponse().GetMessage())
	})

	t.Run("Test 2: Valid config", func(t *testing.T) {
		utils.ClearManagementPlaneResponses(t)
		newConfigFile := "../../config/nginx/nginx-with-test-location.conf"

		err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			newConfigFile,
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		utils.PerformConfigApply(t, nginxInstanceID)

		responses = utils.GetManagementPlaneResponses(t, 2)
		t.Logf("Config apply responses: %v", responses)

		sort.Slice(responses, func(i, j int) bool {
			return responses[i].GetCommandResponse().GetMessage() < responses[j].GetCommandResponse().GetMessage()
		})

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply successful", responses[0].GetCommandResponse().GetMessage())
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
		assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
	})

	t.Run("Test 3: Invalid config", func(t *testing.T) {
		utils.ClearManagementPlaneResponses(t)
		err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			"../../config/nginx/invalid-nginx.conf",
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		utils.PerformConfigApply(t, nginxInstanceID)

		responses = utils.GetManagementPlaneResponses(t, 2)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
		assert.Equal(t, configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
		assert.Equal(t, configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
	})

	t.Run("Test 4: File not in allowed directory", func(t *testing.T) {
		utils.ClearManagementPlaneResponses(t)
		utils.PerformInvalidConfigApply(t, nginxInstanceID)

		responses = utils.GetManagementPlaneResponses(t, 1)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply failed", responses[0].GetCommandResponse().GetMessage())
		assert.Equal(
			t,
			"file not in allowed directories /unknown/nginx.conf",
			responses[0].GetCommandResponse().GetError(),
		)
	})
}
