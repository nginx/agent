// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpc_FileWatcher(t *testing.T) {
	ctx := context.Background()
	teardownTest := utils.SetupConnectionTest(t, true, false, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	defer teardownTest(t)

	utils.VerifyConnection(t, 2, utils.MockManagementPlaneAPIAddress)
	assert.False(t, t.Failed())

	t.Run("Test 1: update nginx config file on data plane", func(t *testing.T) {
		err := utils.Container.CopyFileToContainer(
			ctx,
			"../../config/nginx/nginx-with-server-block-access-log.conf",
			"/etc/nginx/nginx.conf",
			0o666,
		)
		require.NoError(t, err)

		responses := utils.ManagementPlaneResponses(t, 2, utils.MockManagementPlaneAPIAddress)
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
		assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
		assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())

		utils.VerifyUpdateDataPlaneStatus(t, utils.MockManagementPlaneAPIAddress)
	})

	t.Run("Test 2: create new nginx config file", func(t *testing.T) {
		err := utils.Container.CopyFileToContainer(
			ctx,
			"../../config/nginx/empty-nginx.conf",
			"/etc/nginx/test/test.conf",
			0o666,
		)
		require.NoError(t, err)

		responses := utils.ManagementPlaneResponses(t, 3, utils.MockManagementPlaneAPIAddress)
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[2].GetCommandResponse().GetStatus())
		assert.Equal(t, "Successfully updated all files", responses[2].GetCommandResponse().GetMessage())

		utils.VerifyUpdateDataPlaneStatus(t, utils.MockManagementPlaneAPIAddress)
	})

	t.Run("Test 3: delete nginx config file", func(t *testing.T) {
		_, _, err := utils.Container.Exec(
			ctx,
			[]string{"rm", "-rf", "/etc/nginx/test"},
		)
		require.NoError(t, err)

		responses := utils.ManagementPlaneResponses(t, 4, utils.MockManagementPlaneAPIAddress)
		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[3].GetCommandResponse().GetStatus())
		assert.Equal(t, "Successfully updated all files", responses[3].GetCommandResponse().GetMessage())

		utils.VerifyUpdateDataPlaneStatus(t, utils.MockManagementPlaneAPIAddress)
	})
}
