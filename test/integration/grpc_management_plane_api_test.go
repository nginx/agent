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
	teardownTest := SetupConnectionTest(t, false, false)
	defer teardownTest(t)

	timeout := 15 * time.Second

	originalID := VerifyConnection(t, 2)

	stopErr := MockManagementPlaneGrpcContainer.Stop(ctx, &timeout)

	require.NoError(t, stopErr)

	startErr := MockManagementPlaneGrpcContainer.Start(ctx)
	require.NoError(t, startErr)

	ipAddress, err := MockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(t, err)
	ports, err := MockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(t, err)
	MockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)

	currentID := VerifyConnection(t, 2)
	assert.Equal(t, originalID, currentID)
}

// Verify that the agent sends a connection request and an update data plane status request
func TestGrpc_StartUp(t *testing.T) {
	teardownTest := SetupConnectionTest(t, true, false)
	defer teardownTest(t)

	VerifyConnection(t, 2)
	assert.False(t, t.Failed())
	VerifyUpdateDataPlaneHealth(t)
}

func TestGrpc_DataplaneHealthRequest(t *testing.T) {
	teardownTest := SetupConnectionTest(t, true, false)
	defer teardownTest(t)

	VerifyConnection(t, 2)

	responses := GetManagementPlaneResponses(t, 1)
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
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responses = GetManagementPlaneResponses(t, 2)

	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully sent the health status update", responses[1].GetCommandResponse().GetMessage())
}
