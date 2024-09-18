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

const (
	configApplyErrorMessage = "failed validating config NGINX config test failed exit status 1:" +
		" nginx: [emerg] unexpected end of file, expecting \";\" or \"}\" in /etc/nginx/nginx.conf:2\nnginx: " +
		"configuration file /etc/nginx/nginx.conf test failed\n"

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

func setupConnectionTest(tb testing.TB, expectNoErrorsInLogs bool) func(tb testing.TB) {
	tb.Helper()
	ctx := context.Background()

	if os.Getenv("TEST_ENV") == "Container" {
		tb.Log("Running tests in a container environment")

		containerNetwork, err := network.New(
			ctx,
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

		nginxConfPath := "../config/nginx/nginx.conf"
		if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
			nginxConfPath = "../config/nginx/nginx-plus.conf"
		}

		params := &helpers.Parameters{
			NginxConfigPath:      nginxConfPath,
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

		tb.Log("Running tests on local machine")
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

// Verify that the agent sends a connection request and an update data plane status request
func TestGrpc_StartUp(t *testing.T) {
	teardownTest := setupConnectionTest(t, true)
	defer teardownTest(t)

	verifyConnection(t)
	assert.False(t, t.Failed())
	verifyUpdateDataPlaneHealth(t)
}

func TestGrpc_ConfigUpload(t *testing.T) {
	teardownTest := setupConnectionTest(t, true)
	defer teardownTest(t)

	nginxInstanceID := verifyConnection(t)
	assert.False(t, t.Failed())

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
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responses := getManagementPlaneResponses(t, 2)

	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
}

func TestGrpc_ConfigApply(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, false)
	defer teardownTest(t)

	instanceType := "OSS"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		instanceType = "PLUS"
	}

	nginxInstanceID := verifyConnection(t)

	responses := getManagementPlaneResponses(t, 1)
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	t.Run("Test 1: No config changes", func(t *testing.T) {
		performConfigApply(t, nginxInstanceID)

		responses = getManagementPlaneResponses(t, 2)
		t.Logf("Config apply responses: %v", responses)

		assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
		assert.Equal(t, "Config apply successful, no files to change", responses[1].GetCommandResponse().GetMessage())
	})

	t.Run("Test 2: Valid config", func(t *testing.T) {
		err := mockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			"../config/nginx/nginx-with-test-location.conf",
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		performConfigApply(t, nginxInstanceID)

		if instanceType == "OSS" {
			responses = getManagementPlaneResponses(t, 3)
			t.Logf("Config apply responses: %v", responses)

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[2].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply successful", responses[2].GetCommandResponse().GetMessage())
		} else {
			// NGINX Plus contains two extra Successfully updated all files responses as the NginxConfigContext
			// is updated, and the file overview is then updated
			responses = getManagementPlaneResponses(t, 5)
			t.Logf("Config apply responses: %v", responses)
			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[2].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply successful", responses[2].GetCommandResponse().GetMessage())
			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[3].GetCommandResponse().GetStatus())
			assert.Equal(t, "Successfully updated all files", responses[3].GetCommandResponse().GetMessage())
			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[4].GetCommandResponse().GetStatus())
			assert.Equal(t, "Successfully updated all files", responses[4].GetCommandResponse().GetMessage())
		}
	})

	t.Run("Test 3: Invalid config", func(t *testing.T) {
		err := mockManagementPlaneGrpcContainer.CopyFileToContainer(
			ctx,
			"../config/nginx/invalid-nginx.conf",
			fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
			0o666,
		)
		require.NoError(t, err)

		performConfigApply(t, nginxInstanceID)

		if instanceType == "OSS" {
			responses = getManagementPlaneResponses(t, 5)
			t.Logf("Config apply responses: %v", responses)

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[3].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed, rolling back config", responses[3].GetCommandResponse().GetMessage())
			assert.Equal(t, configApplyErrorMessage, responses[3].GetCommandResponse().GetError())
			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[4].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed, rollback successful", responses[4].GetCommandResponse().GetMessage())
			assert.Equal(t, configApplyErrorMessage, responses[4].GetCommandResponse().GetError())
		} else {
			responses = getManagementPlaneResponses(t, 7)
			t.Logf("Config apply responses: %v", len(responses))

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[5].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed, rolling back config", responses[5].GetCommandResponse().GetMessage())
			assert.Equal(t, configApplyErrorMessage, responses[5].GetCommandResponse().GetError())
			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[6].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed, rollback successful", responses[6].GetCommandResponse().GetMessage())
			assert.Equal(t, configApplyErrorMessage, responses[6].GetCommandResponse().GetError())
		}
	})

	t.Run("Test 4: File not in allowed directory", func(t *testing.T) {
		performInvalidConfigApply(t, nginxInstanceID)

		if instanceType == "OSS" {
			responses = getManagementPlaneResponses(t, 6)
			t.Logf("Config apply responses: %v", responses)

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[5].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed", responses[5].GetCommandResponse().GetMessage())
			assert.Equal(
				t,
				"file not in allowed directories /unknown/nginx.conf",
				responses[5].GetCommandResponse().GetError(),
			)
		} else {
			responses = getManagementPlaneResponses(t, 8)
			t.Logf("Config apply responses: %v", responses)

			assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[7].GetCommandResponse().GetStatus())
			assert.Equal(t, "Config apply failed", responses[7].GetCommandResponse().GetMessage())
			assert.Equal(
				t,
				"file not in allowed directories /unknown/nginx.conf",
				responses[7].GetCommandResponse().GetError(),
			)
		}
	})
}

func TestGrpc_FileWatcher(t *testing.T) {
	ctx := context.Background()
	teardownTest := setupConnectionTest(t, true)
	defer teardownTest(t)

	verifyConnection(t)
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

func performConfigApply(t *testing.T, nginxInstanceID string) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/instance/%s/config/apply", mockManagementPlaneAPIAddress, nginxInstanceID)
	resp, err := client.R().EnableTrace().Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func performInvalidConfigApply(t *testing.T, nginxInstanceID string) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)

	body := fmt.Sprintf(`{
			"message_meta": {
				"message_id": "e2254df9-8edd-4900-91ce-88782473bcb9",
				"correlation_id": "9673f3b4-bf33-4d98-ade1-ded9266f6818",
				"timestamp": "2023-01-15T01:30:15.01Z"
			},
			"config_apply_request": {
				"overview": {
					"files": [{
						"file_meta": {
							"name": "/etc/nginx/nginx.conf",
							"hash": "ea57e443-e968-3a50-b842-f37112acde71",
							"modifiedTime": "2023-01-15T01:30:15.01Z",
							"permissions": "0644",
							"size": 0
						},
						"action": "FILE_ACTION_UPDATE"
					}, 
					{
						"file_meta": {
							"name": "/unknown/nginx.conf",
							"hash": "bd1f337d-6874-35ea-9d4d-2b543efd42cf",
							"modifiedTime": "2023-01-15T01:30:15.01Z",
							"permissions": "0644",
							"size": 0
						},
						"action": "FILE_ACTION_ADD"
					}],
					"config_version": {
						"instance_id": "%s",
						"version": "6f343257-55e3-309e-a2eb-bb13af5f80f4"
					}
				}
			}
		}`, nginxInstanceID)
	url := fmt.Sprintf("http://%s/api/v1/requests", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(body).Post(url)
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

func verifyConnection(t *testing.T) string {
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

func verifyUpdateDataPlaneHealth(t *testing.T) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)
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
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		assert.Equal(t, mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS, instances[1].GetInstanceMeta().GetInstanceType())
	} else {
		assert.Equal(t, mpi.InstanceMeta_INSTANCE_TYPE_NGINX, instances[1].GetInstanceMeta().GetInstanceType())
	}
	assert.NotEmpty(t, instances[1].GetInstanceMeta().GetVersion())

	// Verify NGINX instance configuration
	assert.Empty(t, instances[1].GetInstanceConfig().GetActions())
	assert.NotEmpty(t, instances[1].GetInstanceRuntime().GetProcessId())
	assert.Equal(t, "/usr/sbin/nginx", instances[1].GetInstanceRuntime().GetBinaryPath())
	assert.Equal(t, "/etc/nginx/nginx.conf", instances[1].GetInstanceRuntime().GetConfigPath())
}
