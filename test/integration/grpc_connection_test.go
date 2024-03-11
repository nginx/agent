// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/v3/test"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

var (
	mockManagementPlaneGrpcContainer testcontainers.Container
	mockManagementPlaneGrpcAddress   string
	mockManagementPlaneAPIAddress    string
)

type ConnectionRequest struct {
	ConnectionRequest *v1.CreateConnectionRequest `json:"connectionRequest"`
}

func setupConnectionTest(tb testing.TB) func(tb testing.TB) {
	tb.Helper()
	var container testcontainers.Container
	ctx := context.TODO()

	if os.Getenv("TEST_ENV") == "Container" {
		tb.Log("Running tests in a container environment")

		containerNetwork, err := network.New(
			ctx,
			network.WithCheckDuplicate(),
			network.WithAttachable(),
		)
		require.NoError(tb, err)
		tb.Cleanup(func() {
			require.NoError(tb, containerNetwork.Remove(ctx))
		})

		mockManagementPlaneGrpcContainer = test.StartMockManagementPlaneGrpcContainer(
			ctx,
			tb,
			containerNetwork,
		)

		mockManagementPlaneGrpcAddress = "managementPlane:9092"
		tb.Logf("Mock management gRPC server running on %s", mockManagementPlaneGrpcAddress)

		ipAddress, err := mockManagementPlaneGrpcContainer.Host(ctx)
		require.NoError(tb, err)
		ports, err := mockManagementPlaneGrpcContainer.Ports(ctx)
		require.NoError(tb, err)

		mockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)
		tb.Logf("Mock management API server running on %s", mockManagementPlaneAPIAddress)

		container = test.StartContainer(
			ctx,
			tb,
			containerNetwork,
			"../config/nginx/nginx.conf",
			"../config/agent/nginx-config-with-grpc-client.conf",
		)
	} else {
		server := mockGrpc.NewManagementGrpcServer()

		go func(tb testing.TB) {
			tb.Helper()

			listener, err := net.Listen("tcp", "localhost:0")
			assert.NoError(tb, err)

			mockManagementPlaneAPIAddress = listener.Addr().String()

			server.StartServer(listener)
		}(tb)

		go func(tb testing.TB) {
			tb.Helper()

			listener, err := net.Listen("tcp", "localhost:0")
			assert.NoError(tb, err)
			var opts []grpc.ServerOption

			grpcServer := grpc.NewServer(opts...)
			v1.RegisterCommandServiceServer(grpcServer, server)
			err = grpcServer.Serve(listener)
			assert.NoError(tb, err)

			mockManagementPlaneGrpcAddress = listener.Addr().String()
		}(tb)

		tb.Log("Running tests on local machine")
	}

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			test.LogAndTerminateContainers(ctx, tb, mockManagementPlaneGrpcContainer, container)
		}
	}
}

func TestGrpcConnection(t *testing.T) {
	teardownTest := setupConnectionTest(t)
	defer teardownTest(t)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/connection", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var connectionRequest ConnectionRequest

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	// nolint: musttag
	err = json.Unmarshal(responseData, &connectionRequest)
	require.NoError(t, err)

	assert.NotNil(t, connectionRequest.ConnectionRequest)
	assert.Equal(
		t,
		v1.InstanceMeta_INSTANCE_TYPE_AGENT,
		connectionRequest.ConnectionRequest.GetAgent().GetInstanceMeta().GetInstanceType(),
	)
}
