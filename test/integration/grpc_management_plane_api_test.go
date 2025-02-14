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
	"slices"
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

const (
	// configApplyErrorMessage = "failed validating config NGINX config test failed exit status 1:" +
	//	" nginx: [emerg] unexpected end of file, expecting \";\" or \"}\" in /etc/nginx/nginx.conf:2\nnginx: " +
	//	"configuration file /etc/nginx/nginx.conf test failed\n"

	retryCount       = 5
	retryWaitTime    = 2 * time.Second
	retryMaxWaitTime = 3 * time.Second
)

var (
	container                        testcontainers.Container
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

func setupConnectionTest(tb testing.TB, expectNoErrorsInLogs, nginxless bool) func(tb testing.TB) {
	tb.Helper()
	ctx := context.Background()

	if os.Getenv("TEST_ENV") == "Container" {
		setupContainerEnvironment(ctx, tb, nginxless)
	} else {
		setupLocalEnvironment(tb)
	}

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			helpers.LogAndTerminateContainers(
				ctx,
				tb,
				mockManagementPlaneGrpcContainer,
				container,
				expectNoErrorsInLogs,
			)
		}
	}
}

// setupContainerEnvironment sets up the container environment for testing.
func setupContainerEnvironment(ctx context.Context, tb testing.TB, nginxless bool) {
	tb.Helper()
	tb.Log("Running tests in a container environment")

	containerNetwork := createContainerNetwork(ctx, tb)
	setupMockManagementPlaneGrpc(ctx, tb, containerNetwork)

	params := &helpers.Parameters{
		NginxAgentConfigPath: "../config/agent/nginx-config-with-grpc-client.conf",
		LogMessage:           "Agent connected",
	}
	switch nginxless {
	case true:
		container = helpers.StartNginxLessContainer(ctx, tb, containerNetwork, params)
	case false:
		setupNginxContainer(ctx, tb, containerNetwork, params)
	}
}

// createContainerNetwork creates and configures a container network.
func createContainerNetwork(ctx context.Context, tb testing.TB) *testcontainers.DockerNetwork {
	tb.Helper()
	containerNetwork, err := network.New(ctx, network.WithAttachable())
	require.NoError(tb, err)
	tb.Cleanup(func() {
		networkErr := containerNetwork.Remove(ctx)
		tb.Logf("Error removing container network: %v", networkErr)
	})

	return containerNetwork
}

// setupMockManagementPlaneGrpc initializes the mock management plane gRPC container.
func setupMockManagementPlaneGrpc(ctx context.Context, tb testing.TB, containerNetwork *testcontainers.DockerNetwork) {
	tb.Helper()
	mockManagementPlaneGrpcContainer = helpers.StartMockManagementPlaneGrpcContainer(ctx, tb, containerNetwork)
	mockManagementPlaneGrpcAddress = "managementPlane:9092"
	tb.Logf("Mock management gRPC server running on %s", mockManagementPlaneGrpcAddress)

	ipAddress, err := mockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(tb, err)
	ports, err := mockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(tb, err)

	mockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)
	tb.Logf("Mock management API server running on %s", mockManagementPlaneAPIAddress)
}

// setupNginxContainer configures and starts the NGINX container.
func setupNginxContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	params *helpers.Parameters,
) {
	tb.Helper()
	nginxConfPath := "../config/nginx/nginx.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConfPath = "../config/nginx/nginx-plus.conf"
	}
	params.NginxConfigPath = nginxConfPath

	container = helpers.StartContainer(ctx, tb, containerNetwork, params)
}

// setupLocalEnvironment configures the local testing environment.
func setupLocalEnvironment(tb testing.TB) {
	tb.Helper()
	tb.Log("Running tests on local machine")

	requestChan := make(chan *mpi.ManagementPlaneRequest)
	server := mockGrpc.NewCommandService(requestChan, os.TempDir())

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
}

func TestGrpc_ConfigApply(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, false, false)
	defer teardownTest(t)

	instanceType := "OSS"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		instanceType = "PLUS"
	}

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

		if instanceType == "OSS" {
			responses = getManagementPlaneResponses(t, 1)
			t.Logf("Config apply responses: %v", responses)

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply successful", responses[0].GetCommandResponse().GetMessage())
		} else {
			// NGINX Plus contains two extra Successfully updated all files responses as the NginxConfigContext
			// is updated, and the file overview is then updated
			time.Sleep(5 * time.Second)

			responses = getManagementPlaneResponses(t, 3)
			t.Logf("Config apply responses: %v", responses)
			//assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
			//assert.Equal(t, "Config apply successful", responses[0].GetCommandResponse().GetMessage())
			//assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
			//assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
			//assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[2].GetCommandResponse().GetStatus())
			//assert.Equal(t, "Successfully updated all files", responses[2].GetCommandResponse().GetMessage())
		}
	})

	// t.Run("Test 3: Invalid config", func(t *testing.T) {
	//	clearManagementPlaneResponses(t)
	//	err := mockManagementPlaneGrpcContainer.CopyFileToContainer(
	//		ctx,
	//		"../config/nginx/invalid-nginx.conf",
	//		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
	//		0o666,
	//	)
	//	require.NoError(t, err)
	//
	//	performConfigApply(t, nginxInstanceID)
	//
	//	if instanceType == "OSS" {
	//		responses = getManagementPlaneResponses(t, 2)
	//		t.Logf("Config apply responses: %v", responses)
	//
	// assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
	// assert.Equal(t, "Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
	// assert.Equal(t, configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
	// assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
	//	assert.Equal(t, "Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
	//	assert.Equal(t, configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
	//	} else {
	//		responses = getManagementPlaneResponses(t, 2)
	//		t.Logf("Config apply responses: %v", len(responses))
	//
	//		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
	//	assert.Equal(t, "Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
	//		assert.Equal(t, configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
	//		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
	//	assert.Equal(t, "Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
	//		assert.Equal(t, configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
	//	}
	// })
	//
	// t.Run("Test 4: File not in allowed directory", func(t *testing.T) {
	//	clearManagementPlaneResponses(t)
	//	performInvalidConfigApply(t, nginxInstanceID)
	//
	//	responses = getManagementPlaneResponses(t, 1)
	//	t.Logf("Config apply responses: %v", responses)
	//
	//	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[0].GetCommandResponse().GetStatus())
	//	assert.Equal(t, "Config apply failed", responses[0].GetCommandResponse().GetMessage())
	//	assert.Equal(
	//		t,
	//		"file not in allowed directories /unknown/nginx.conf",
	//		responses[0].GetCommandResponse().GetError(),
	//	)
	// })
}

func performConfigApply(t *testing.T, nginxInstanceID string) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/instance/%s/config/apply", mockManagementPlaneAPIAddress, nginxInstanceID)
	resp, err := client.R().EnableTrace().Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func getManagementPlaneResponses(t *testing.T, numberOfExpectedResponses int) []*mpi.DataPlaneResponse {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			responseData := r.Body()
			assert.True(t, json.Valid(responseData))

			response := []*mpi.DataPlaneResponse{}
			unmarshalErr := json.Unmarshal(responseData, &response)
			require.NoError(t, unmarshalErr)

			return len(response) != numberOfExpectedResponses || r.StatusCode() == http.StatusNotFound
		},
	)

	url := fmt.Sprintf("http://%s/api/v1/responses", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()
	t.Logf("Responses: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	response := []*mpi.DataPlaneResponse{}
	unmarshalErr := json.Unmarshal(responseData, &response)
	require.NoError(t, unmarshalErr)

	assert.Len(t, response, numberOfExpectedResponses)

	slices.SortFunc(response, func(a, b *mpi.DataPlaneResponse) int {
		return a.GetMessageMeta().GetTimestamp().AsTime().Compare(b.GetMessageMeta().GetTimestamp().AsTime())
	})

	return response
}

func clearManagementPlaneResponses(t *testing.T) {
	t.Helper()

	client := resty.New()

	url := fmt.Sprintf("http://%s/api/v1/responses", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Delete(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func verifyConnection(t *testing.T, instancesLength int) string {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

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

	assert.Len(t, resource.GetInstances(), instancesLength)

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
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS:
			nginxInstanceMeta := instance.GetInstanceMeta()

			nginxInstanceID = nginxInstanceMeta.GetInstanceId()
			assert.NotEmpty(t, nginxInstanceID)
			assert.NotEmpty(t, nginxInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx/nginx.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_UNIT,
			mpi.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED:
			fallthrough
		default:
			t.Fail()
		}
	}

	return nginxInstanceID
}
