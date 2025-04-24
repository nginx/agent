// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpc_FileWatcher(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, true, false)
	defer teardownTest(t)

	verifyConnection(t, 2)
	assert.False(t, t.Failed())

	err := container.CopyFileToContainer(
		ctx,
		"../config/nginx/nginx-with-server-block-access-log.conf",
		"/etc/nginx/nginx.conf",
		0o666,
	)
	require.NoError(t, err)

	responses := getManagementPlaneResponses(t, 2)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())

	verifyUpdateDataPlaneStatus(t)
}
