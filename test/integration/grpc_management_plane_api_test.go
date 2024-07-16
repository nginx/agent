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
	"sort"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
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
		ConnectionRequest *mpi.CreateConnectionRequest `json:"connectionRequest"`
	}
	Instance struct {
		InstanceMeta    *mpi.InstanceMeta    `json:"instance_meta"`
		InstanceRuntime *mpi.InstanceRuntime `json:"instance_runtime"`
	}
	NginxUpdateDataPlaneHealthRequest struct {
		MessageMeta *mpi.MessageMeta `json:"message_meta"`
		Instances   []Instance       `json:"instances"`
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
		requestChan := make(chan *mpi.ManagementPlaneRequest)
		server := mockGrpc.NewCommandService(requestChan)

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
			mpi.RegisterCommandServiceServer(grpcServer, server)
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

func TestGrpc_ConfigUpload(t *testing.T) {
	teardownTest := setupConnectionTest(t)
	defer teardownTest(t)

	nginxInstanceID := verifyConnection(t)

	request := fmt.Sprintf(`{
	"message_meta": {
		"message_id": "5d0fa83e-351c-4009-90cd-1f2acce2d184",
		"correlation_id": "79794c1c-8e91-47c1-a92c-b9a0c3f1a263",
		"timestamp": "2023-01-15T01:30:15.01Z"
	},
	"config_upload_request": {
	"instance_id": "%s"
	}
}`, nginxInstanceID)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/requests", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	client = resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(1 * time.Second).SetRetryMaxWaitTime(3 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return len(r.Body()) == 0 || r.StatusCode() == http.StatusNotFound
		},
	)

	url = fmt.Sprintf("http://%s/api/v1/responses", mockManagementPlaneAPIAddress)
	resp, err = client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	response := []*mpi.DataPlaneResponse{}
	unmarshalErr := json.Unmarshal(responseData, &response)
	require.NoError(t, unmarshalErr)

	assert.Len(t, response, 1)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, response[0].GetCommandResponse().GetStatus())
}

func verifyConnection(t *testing.T) string {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/connection", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	connectionRequest := mpi.CreateConnectionRequest{}

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

	assert.Len(t, resource.GetInstances(), 2)

	var nginxInstanceID string

	for _, instance := range resource.GetInstances() {
		switch instance.GetInstanceMeta().GetInstanceType() {
		case mpi.InstanceMeta_INSTANCE_TYPE_AGENT:
			agentInstanceMeta := instance.GetInstanceMeta()

			assert.NotEmpty(t, agentInstanceMeta.GetInstanceId())
			assert.NotEmpty(t, agentInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx-agent/nginx-agent.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX:
			nginxInstanceMeta := instance.GetInstanceMeta()

			nginxInstanceID = nginxInstanceMeta.GetInstanceId()
			assert.NotEmpty(t, nginxInstanceID)
			assert.NotEmpty(t, nginxInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx/nginx.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS,
			mpi.InstanceMeta_INSTANCE_TYPE_UNIT,
			mpi.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED:
			fallthrough
		default:
			t.Fail()
		}
	}

	return nginxInstanceID
}

func verifyUpdateDataPlaneStatus(t *testing.T) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/status", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	updateDataPlaneStatusRequest := mpi.UpdateDataPlaneStatusRequest{}

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
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].GetInstanceMeta().GetInstanceType() < instances[j].GetInstanceMeta().GetInstanceType()
	})
	assert.Len(t, instances, 2)

	// Verify agent instance metadata
	assert.NotEmpty(t, instances[0].GetInstanceMeta().GetInstanceId())
	assert.Equal(t, mpi.InstanceMeta_INSTANCE_TYPE_AGENT, instances[0].GetInstanceMeta().GetInstanceType())
	assert.NotEmpty(t, instances[0].GetInstanceMeta().GetVersion())

	// Verify agent instance configuration
	assert.Empty(t, instances[0].GetInstanceConfig().GetActions())
	assert.NotEmpty(t, instances[0].GetInstanceRuntime().GetProcessId())
	assert.Equal(t, "/usr/bin/nginx-agent", instances[0].GetInstanceRuntime().GetBinaryPath())
	assert.Equal(t, "/etc/nginx-agent/nginx-agent.conf", instances[0].GetInstanceRuntime().GetConfigPath())

	// Verify NGINX instance metadata
	assert.NotEmpty(t, instances[1].GetInstanceMeta().GetInstanceId())
	assert.Equal(t, mpi.InstanceMeta_INSTANCE_TYPE_NGINX, instances[1].GetInstanceMeta().GetInstanceType())
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
	client.SetRetryCount(3).SetRetryWaitTime(5 * time.Second).SetRetryMaxWaitTime(15 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return r.StatusCode() == http.StatusNotFound
		},
	)

	url := fmt.Sprintf("http://%s/api/v1/health", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}

	updateDataPlaneHealthRequest := mpi.UpdateDataPlaneHealthRequest{}
	unmarshalErr := pb.Unmarshal(responseData, &updateDataPlaneHealthRequest)
	require.NoError(t, unmarshalErr)

	t.Logf("UpdateDataPlaneHealthRequest: %v", &updateDataPlaneHealthRequest)

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
	assert.Equal(t, mpi.InstanceHealth_INSTANCE_HEALTH_STATUS_HEALTHY, healths[0].GetInstanceHealthStatus())
}
