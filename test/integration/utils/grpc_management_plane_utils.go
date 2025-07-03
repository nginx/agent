// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package utils

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

var (
	Container                                 testcontainers.Container
	MockManagementPlaneGrpcContainer          testcontainers.Container
	AuxiliaryMockManagementPlaneGrpcContainer testcontainers.Container
	MockManagementPlaneGrpcAddress            string
	AuxiliaryMockManagementPlaneGrpcAddress   string
)

const (
	instanceLen      = 2
	statusRetryCount = 3
	retryWait        = 50 * time.Millisecond
	retryMaxWait     = 200 * time.Millisecond
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

func SetupConnectionTest(tb testing.TB, expectNoErrorsInLogs, nginxless, auxiliaryServer bool,
	agentConfig string,
) func(tb testing.TB) {
	tb.Helper()
	ctx := context.Background()

	if os.Getenv("TEST_ENV") == "Container" {
		setupContainerEnvironment(ctx, tb, nginxless, auxiliaryServer, agentConfig)
	} else {
		setupLocalEnvironment(tb)
	}

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			helpers.LogAndTerminateContainers(
				ctx,
				tb,
				MockManagementPlaneGrpcContainer,
				Container,
				expectNoErrorsInLogs,
				AuxiliaryMockManagementPlaneGrpcContainer,
			)
		}
	}
}

// setupContainerEnvironment sets up the container environment for testing.
// nolint: revive
func setupContainerEnvironment(ctx context.Context, tb testing.TB, nginxless, auxiliaryServer bool,
	agentConfig string,
) {
	tb.Helper()
	tb.Log("Running tests in a container environment")

	containerNetwork := createContainerNetwork(ctx, tb)
	setupMockManagementPlaneGrpc(ctx, tb, containerNetwork)
	if auxiliaryServer {
		setupAuxiliaryMockManagementPlaneGrpc(ctx, tb, containerNetwork)
	}

	params := &helpers.Parameters{
		NginxAgentConfigPath: agentConfig,
		LogMessage:           "Agent connected",
	}
	switch nginxless {
	case true:
		Container = helpers.StartNginxLessContainer(ctx, tb, containerNetwork, params)
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
	MockManagementPlaneGrpcContainer = helpers.StartMockManagementPlaneGrpcContainer(ctx, tb, containerNetwork)
	MockManagementPlaneGrpcAddress = "managementPlane:9092"
	tb.Logf("Mock management gRPC server running on %s", MockManagementPlaneGrpcAddress)

	ipAddress, err := MockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(tb, err)
	ports, err := MockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(tb, err)

	MockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)
	tb.Logf("Mock management API server running on %s", MockManagementPlaneAPIAddress)
}

func setupAuxiliaryMockManagementPlaneGrpc(ctx context.Context, tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
) {
	tb.Helper()
	AuxiliaryMockManagementPlaneGrpcContainer = helpers.StartAuxiliaryMockManagementPlaneGrpcContainer(ctx,
		tb, containerNetwork)
	AuxiliaryMockManagementPlaneGrpcAddress = "managementPlaneAuxiliary:9095"
	tb.Logf("Auxiliary mock management gRPC server running on %s", AuxiliaryMockManagementPlaneGrpcAddress)

	ipAddress, err := AuxiliaryMockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(tb, err)
	ports, err := AuxiliaryMockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(tb, err)

	AuxiliaryMockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9096/tcp"][0].HostPort)
	tb.Logf("Auxiliary mock management API server running on %s", AuxiliaryMockManagementPlaneAPIAddress)
}

// setupNginxContainer configures and starts the NGINX container.
func setupNginxContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	params *helpers.Parameters,
) {
	tb.Helper()
	nginxConfPath := "../../config/nginx/nginx.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConfPath = "../../config/nginx/nginx-plus.conf"
	}
	params.NginxConfigPath = nginxConfPath

	Container = helpers.StartContainer(ctx, tb, containerNetwork, params)
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

		MockManagementPlaneAPIAddress = listener.Addr().String()

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

		MockManagementPlaneGrpcAddress = listener.Addr().String()
	}(tb)
}

func ManagementPlaneResponses(t *testing.T, numberOfExpectedResponses int,
	mockManagementPlaneAPIAddress string,
) []*mpi.DataPlaneResponse {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)
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

func ClearManagementPlaneResponses(t *testing.T, mockManagementPlaneAPIAddress string) {
	t.Helper()

	client := resty.New()

	url := fmt.Sprintf("http://%s/api/v1/responses", mockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().Delete(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func VerifyConnection(t *testing.T, instancesLength int, mockManagementPlaneAPIAddress string) string {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)
	connectionRequest := mpi.CreateConnectionRequest{}
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			responseData := r.Body()

			pb := protojson.UnmarshalOptions{DiscardUnknown: true}
			unmarshalErr := pb.Unmarshal(responseData, &connectionRequest)

			return r.StatusCode() == http.StatusNotFound || unmarshalErr != nil
		},
	)
	url := fmt.Sprintf("http://%s/api/v1/connection", mockManagementPlaneAPIAddress)
	t.Logf("Connecting to %s", url)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

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
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX_APP_PROTECT:
			instanceMeta := instance.GetInstanceMeta()
			assert.NotEmpty(t, instanceMeta.GetInstanceId())
			assert.NotEmpty(t, instanceMeta.GetVersion())

			instanceRuntimeInfo := instance.GetInstanceRuntime().GetNginxAppProtectRuntimeInfo()
			assert.NotEmpty(t, instanceRuntimeInfo.GetRelease())
			assert.NotEmpty(t, instanceRuntimeInfo.GetAttackSignatureVersion())
			assert.NotEmpty(t, instanceRuntimeInfo.GetThreatCampaignVersion())
		case mpi.InstanceMeta_INSTANCE_TYPE_UNIT,
			mpi.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED:
			fallthrough
		default:
			t.Fail()
		}
	}

	return nginxInstanceID
}

func VerifyUpdateDataPlaneHealth(t *testing.T, mockManagementPlaneAPIAddress string) {
	t.Helper()

	client := resty.New()

	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

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

func VerifyUpdateDataPlaneStatus(t *testing.T, mockManagementPlaneAPIAddress string) {
	t.Helper()

	client := resty.New()

	client.SetRetryCount(statusRetryCount).SetRetryWaitTime(retryWait).SetRetryMaxWaitTime(retryMaxWait)

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

	assert.Len(t, instances, instanceLen)

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
