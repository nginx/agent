// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpc_Reconnection(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, false, false)
	defer teardownTest(t)

	timeout := 15 * time.Second

	originalID := verifyConnection(t, 2)

	stopErr := mockManagementPlaneGrpcContainer.Stop(ctx, &timeout)

	require.NoError(t, stopErr)

	startErr := mockManagementPlaneGrpcContainer.Start(ctx)
	require.NoError(t, startErr)

	ipAddress, err := mockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(t, err)
	ports, err := mockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(t, err)
	mockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)

	currentID := verifyConnection(t, 2)
	assert.Equal(t, originalID, currentID)
}

// Verify that the agent sends a connection request and an update data plane status request
func TestGrpc_StartUp(t *testing.T) {
	teardownTest := setupConnectionTest(t, true, false)
	defer teardownTest(t)

	verifyConnection(t, 2)
	assert.False(t, t.Failed())
	verifyUpdateDataPlaneHealth(t)
}

func TestGrpc_DataplaneHealthRequest(t *testing.T) {
	teardownTest := setupConnectionTest(t, true, false)
	defer teardownTest(t)

	verifyConnection(t, 2)

	responses := getManagementPlaneResponses(t, 1)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	assert.False(t, t.Failed())

	request := `{
			"message_meta": {
				"message_id": "5d0fa83e-351c-4009-90cd-1f2acce2d184",
				"correlation_id": "79794c1c-8e91-47c1-a92c-b9a0c3f1a263",
				"timestamp": "2023-01-15T01:30:15.01Z"
			},
			"health_request": {}
		}`

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responses = getManagementPlaneResponses(t, 2)

	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully sent the health status update", responses[1].GetCommandResponse().GetMessage())
}
