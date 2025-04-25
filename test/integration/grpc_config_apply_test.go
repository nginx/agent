// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"fmt"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configApplyErrorMessage = "failed validating config NGINX config test failed exit status 1:" +
		" nginx: [emerg] unexpected end of file, expecting \";\" or \"}\" in /etc/nginx/nginx.conf:2\nnginx: " +
		"configuration file /etc/nginx/nginx.conf test failed\n"
)

func TestGrpc_ConfigApply(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, false, false)
	defer teardownTest(t)

	nginxInstanceID := verifyConnection(t, 2)

	responses := getManagementPlaneResponses(t, 1)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	t.Run("Test 1: No config changes", func(t *testing.T) {
		clearManagementPlaneResponses(t)
		performConfigApply(t, nginxInstanceID)
		responses = getManagementPlaneResponses(t, 1)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply successful, no files to change", responses[0].GetCommandResponse().GetMessage())
	})

	t.Run("Test 2: Valid config", func(t *testing.T) {
		clearManagementPlaneResponses(t)
		err := mockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			"../config/nginx/nginx-with-test-location.conf",
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		performConfigApply(t, nginxInstanceID)

		responses = getManagementPlaneResponses(t, 1)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply successful", responses[0].GetCommandResponse().GetMessage())
	})

	t.Run("Test 3: Invalid config", func(t *testing.T) {
		clearManagementPlaneResponses(t)
		err := mockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			"../config/nginx/invalid-nginx.conf",
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		performConfigApply(t, nginxInstanceID)

		responses = getManagementPlaneResponses(t, 2)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
		assert.Equal(t, configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
		assert.Equal(t, configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
	})

	t.Run("Test 4: File not in allowed directory", func(t *testing.T) {
		clearManagementPlaneResponses(t)
		performInvalidConfigApply(t, nginxInstanceID)

		responses = getManagementPlaneResponses(t, 1)
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
