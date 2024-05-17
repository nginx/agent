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
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	mockManagementPlaneGrpcContainer testcontainers.Container
	mockManagementPlaneGrpcAddress   string
	mockManagementPlaneAPIAddress    string
)

type (
	ConnectionRequest struct {
		ConnectionRequest *v1.CreateConnectionRequest `json:"connectionRequest"`
	}
	Instance struct {
		InstanceMeta    *v1.InstanceMeta    `json:"instance_meta"`
		InstanceRuntime *v1.InstanceRuntime `json:"instance_runtime"`
	}
	NginxUpdateDataPlaneHealthRequest struct {
		MessageMeta *v1.MessageMeta `json:"message_meta"`
		Instances   []Instance      `json:"instances"`
	}
	UpdateDataPlaneStatusRequest struct {
		UpdateDataPlaneStatusRequest NginxUpdateDataPlaneHealthRequest `json:"updateDataPlaneStatusRequest"`
	}
)

func setupConnectionTest(tb testing.TB) func(tb testing.TB) {
	tb.Helper()
	var container testcontainers.Container
	ctx := context.Background()

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

		mockManagementPlaneGrpcContainer = helpers.StartMockManagementPlaneGrpcContainer(
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

		params := &helpers.Parameters{
			NginxConfigPath:      "../config/nginx/nginx.conf",
			NginxAgentConfigPath: "../config/agent/nginx-config-with-grpc-client.conf",
			LogMessage:           "Agent connected",
		}

		container = helpers.StartContainer(
			ctx,
			tb,
			containerNetwork,
			params,
		)
	} else {
		server := mockGrpc.NewCommandService()

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
			helpers.LogAndTerminateContainers(ctx, tb, mockManagementPlaneGrpcContainer, container)
		}
	}
}

// Verify that the agent sends a connection request and an update data plane status request
func TestGrpc_StartUp(t *testing.T) {
	teardownTest := setupConnectionTest(t)
	defer teardownTest(t)

	verifyConnection(t)
	verifyUpdateDataPlaneStatus(t)
	verifyUpdateDataPlaneHealth(t)
}

func verifyConnection(t *testing.T) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/connection", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	connectionRequest := v1.CreateConnectionRequest{}

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	unmarshalErr := pb.Unmarshal(responseData, &connectionRequest)
	require.NoError(t, unmarshalErr)

	t.Logf("ConnectionRequest: %v", &connectionRequest)

	resource := connectionRequest.GetResource()

	assert.NotNil(t, resource.GetResourceId())
	assert.NotNil(t, resource.GetContainerInfo().GetContainerId())

	agentInstance := resource.GetInstances()[0]
	agentInstanceMeta := agentInstance.GetInstanceMeta()

	assert.NotEmpty(t, agentInstanceMeta.GetInstanceId())
	assert.NotEmpty(t, agentInstanceMeta.GetVersion())

	assert.NotEmpty(t, agentInstance.GetInstanceRuntime().GetBinaryPath())

	assert.Equal(t, v1.InstanceMeta_INSTANCE_TYPE_AGENT, agentInstanceMeta.GetInstanceType())
	assert.Equal(t, "/etc/nginx-agent/nginx-agent.conf", agentInstance.GetInstanceRuntime().GetConfigPath())

	nginxInstance := resource.GetInstances()[1]
	nginxInstanceMeta := nginxInstance.GetInstanceMeta()

	assert.NotEmpty(t, nginxInstanceMeta.GetInstanceId())
	assert.NotEmpty(t, nginxInstanceMeta.GetVersion())

	assert.NotEmpty(t, nginxInstance.GetInstanceRuntime().GetBinaryPath())

	assert.Equal(t, v1.InstanceMeta_INSTANCE_TYPE_NGINX, nginxInstanceMeta.GetInstanceType())
	assert.Equal(t, "/etc/nginx/nginx.conf", nginxInstance.GetInstanceRuntime().GetConfigPath())
}

func verifyUpdateDataPlaneStatus(t *testing.T) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/status", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	updateDataPlaneStatusRequest := v1.UpdateDataPlaneStatusRequest{}

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	unmarshalErr := pb.Unmarshal(responseData, &updateDataPlaneStatusRequest)
	require.NoError(t, unmarshalErr)

	t.Logf("UpdateDataPlaneStatusRequest: %v", &updateDataPlaneStatusRequest)

	assert.NotNil(t, &updateDataPlaneStatusRequest)

	// Verify message metadata
	messageMeta := updateDataPlaneStatusRequest.GetMessageMeta()
	assert.NotEmpty(t, messageMeta.GetCorrelationId())
	assert.NotEmpty(t, messageMeta.GetMessageId())
	assert.NotEmpty(t, messageMeta.GetTimestamp())

	instances := updateDataPlaneStatusRequest.GetResource().GetInstances()
	assert.Len(t, instances, 2)

	// Verify agent instance metadata
	assert.NotEmpty(t, instances[0].GetInstanceMeta().GetInstanceId())
	assert.Equal(t, v1.InstanceMeta_INSTANCE_TYPE_AGENT, instances[0].GetInstanceMeta().GetInstanceType())
	assert.NotEmpty(t, instances[0].GetInstanceMeta().GetVersion())

	// Verify agent instance configuration
	assert.Empty(t, instances[0].GetInstanceConfig().GetActions())
	assert.NotEmpty(t, instances[0].GetInstanceRuntime().GetProcessId())
	assert.Equal(t, "/usr/bin/nginx-agent", instances[0].GetInstanceRuntime().GetBinaryPath())
	assert.Equal(t, "/etc/nginx-agent/nginx-agent.conf", instances[0].GetInstanceRuntime().GetConfigPath())

	// Verify NGINX instance metadata
	assert.NotEmpty(t, instances[1].GetInstanceMeta().GetInstanceId())
	assert.Equal(t, v1.InstanceMeta_INSTANCE_TYPE_NGINX, instances[1].GetInstanceMeta().GetInstanceType())
	assert.NotEmpty(t, instances[1].GetInstanceMeta().GetVersion())

	// Verify NGINX instance configuration
	assert.Empty(t, instances[1].GetInstanceConfig().GetActions())
	assert.NotEmpty(t, instances[1].GetInstanceRuntime().GetProcessId())
	assert.Equal(t, "/usr/sbin/nginx", instances[1].GetInstanceRuntime().GetBinaryPath())
	assert.Equal(t, "/etc/nginx/nginx.conf", instances[1].GetInstanceRuntime().GetConfigPath())
}

func verifyUpdateDataPlaneHealth(t *testing.T) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/health", mockManagementPlaneAPIAddress)

	resp, err := client.R().EnableTrace().Get(url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}

	updateDataPlaneHealthRequest := v1.UpdateDataPlaneHealthRequest{}
	unmarshalErr := pb.Unmarshal(responseData, &updateDataPlaneHealthRequest)
	require.NoError(t, unmarshalErr)

	t.Logf("UpdateDataPlaneStatusRequest: %v", &updateDataPlaneHealthRequest)

	assert.NotNil(t, &updateDataPlaneHealthRequest)

	// Verify message metadata
	messageMeta := updateDataPlaneHealthRequest.GetMessageMeta()
	assert.NotEmpty(t, messageMeta.GetCorrelationId())
	assert.NotEmpty(t, messageMeta.GetMessageId())
	assert.NotEmpty(t, messageMeta.GetTimestamp())

	healths := updateDataPlaneHealthRequest.GetInstanceHealths()
	assert.Len(t, healths, 1)

	// Verify health metadata
	assert.NotEmpty(t, healths[0].GetInstanceId())
	assert.Equal(t, v1.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY, healths[0].GetInstanceHealthStatus())
	assert.NotEmpty(t, healths[0].GetDescription())
}
