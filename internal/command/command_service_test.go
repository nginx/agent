// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/stub"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FakeSubscribeClient struct {
	grpc.ClientStream
}

func (*FakeSubscribeClient) Send(*mpi.DataPlaneResponse) error {
	return nil
}

// nolint: nilnil
func (*FakeSubscribeClient) Recv() (*mpi.ManagementPlaneRequest, error) {
	return nil, nil
}

func TestCommandService_NewCommandService(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	defer commandService.CancelSubscription(ctx)

	assert.Eventually(
		t,
		func() bool { return commandServiceClient.SubscribeCallCount() > 0 },
		2*time.Second,
		10*time.Millisecond,
	)
}

func TestCommandService_UpdateDataPlaneStatus(t *testing.T) {
	ctx := context.Background()

	fakeSubscribeClient := &FakeSubscribeClient{}

	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	commandServiceClient.SubscribeReturns(fakeSubscribeClient, nil)

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)
	defer commandService.CancelSubscription(ctx)

	// Fail first time since there are no other instances besides the agent
	err := commandService.UpdateDataPlaneStatus(ctx, protos.GetHostResource())
	require.Error(t, err)

	resource := protos.GetHostResource()
	resource.Instances = append(resource.Instances, protos.GetNginxOssInstance([]string{}))
	_, connectionErr := commandService.CreateConnection(ctx, resource)
	require.NoError(t, connectionErr)
	err = commandService.UpdateDataPlaneStatus(ctx, resource)

	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.UpdateDataPlaneStatusCallCount())
}

func TestCommandService_UpdateDataPlaneStatusSubscribeError(t *testing.T) {
	correlationID, _ := helpers.CreateTestIDs(t)
	ctx := context.WithValue(
		context.Background(),
		logger.CorrelationIDContextKey,
		slog.Any(logger.CorrelationIDKey, correlationID.String()),
	)

	fakeSubscribeClient := &FakeSubscribeClient{}

	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	commandServiceClient.SubscribeReturns(fakeSubscribeClient, errors.New("sub error"))
	commandServiceClient.UpdateDataPlaneStatusReturns(nil, errors.New("ret error"))

	logBuf := &bytes.Buffer{}
	stub.StubLoggerWith(logBuf)

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)
	defer commandService.CancelSubscription(ctx)

	commandService.isConnected.Store(true)

	err := commandService.UpdateDataPlaneStatus(ctx, protos.GetHostResource())
	require.Error(t, err)

	helpers.ValidateLog(t, "Failed to send update data plane status", logBuf)

	logBuf.Reset()
}

func TestCommandService_CreateConnection(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	// connection created when no nginx instance found
	resource := protos.GetHostResource()
	_, err := commandService.CreateConnection(ctx, resource)
	require.NoError(t, err)
}

func TestCommandService_UpdateDataPlaneHealth(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	// connection not created yet
	err := commandService.UpdateDataPlaneHealth(ctx, protos.GetInstanceHealths())

	require.Error(t, err)
	assert.Equal(t, 0, commandServiceClient.UpdateDataPlaneHealthCallCount())

	// connection created
	resource := protos.GetHostResource()
	resource.Instances = append(resource.Instances, protos.GetNginxOssInstance([]string{}))
	_, err = commandService.CreateConnection(ctx, resource)
	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.CreateConnectionCallCount())

	err = commandService.UpdateDataPlaneHealth(ctx, protos.GetInstanceHealths())

	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.UpdateDataPlaneHealthCallCount())
}

func TestCommandService_SendDataPlaneResponse(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	subscribeClient := &FakeSubscribeClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	commandService.subscribeClientMutex.Lock()
	commandService.subscribeClient = subscribeClient
	commandService.subscribeClientMutex.Unlock()

	err := commandService.SendDataPlaneResponse(ctx, protos.OKDataPlaneResponse())

	require.NoError(t, err)
}

func TestCommandService_isValidRequest(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	subscribeClient := &FakeSubscribeClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	commandService.subscribeClientMutex.Lock()
	commandService.subscribeClient = subscribeClient
	commandService.subscribeClientMutex.Unlock()

	nginxInstance := protos.GetNginxOssInstance([]string{})

	commandService.instances = append(commandService.instances, nginxInstance)

	testCases := []struct {
		req    *mpi.ManagementPlaneRequest
		name   string
		result bool
	}{
		{
			name: "Test 1: valid health request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request:     &mpi.ManagementPlaneRequest_HealthRequest{HealthRequest: &mpi.HealthRequest{}},
			},
			result: true,
		},
		{
			name: "Test 2: valid config apply request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: protos.CreateConfigApplyRequest(&mpi.FileOverview{
						Files: make([]*mpi.File, 0),
						ConfigVersion: &mpi.ConfigVersion{
							InstanceId: nginxInstance.GetInstanceMeta().GetInstanceId(),
							Version:    "e23brbei3u2bru93",
						},
					}),
				},
			},
			result: true,
		},
		{
			name: "Test 3: invalid config apply request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: protos.CreateConfigApplyRequest(&mpi.FileOverview{
						Files: make([]*mpi.File, 0),
						ConfigVersion: &mpi.ConfigVersion{
							InstanceId: "unknown-id",
							Version:    "e23brbei3u2bru93",
						},
					}),
				},
			},
			result: false,
		},
		{
			name: "Test 4: valid config upload request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
					ConfigUploadRequest: &mpi.ConfigUploadRequest{
						Overview: &mpi.FileOverview{
							Files: make([]*mpi.File, 0),
							ConfigVersion: &mpi.ConfigVersion{
								InstanceId: nginxInstance.GetInstanceMeta().GetInstanceId(),
								Version:    "e23brbei3u2bru93",
							},
						},
					},
				},
			},
			result: true,
		},
		{
			name: "Test 5: invalid config upload request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
					ConfigUploadRequest: &mpi.ConfigUploadRequest{
						Overview: &mpi.FileOverview{
							Files: make([]*mpi.File, 0),
							ConfigVersion: &mpi.ConfigVersion{
								InstanceId: "unknown-id",
								Version:    "e23brbei3u2bru93",
							},
						},
					},
				},
			},
			result: false,
		},
		{
			name: "Test 6: valid action request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ActionRequest{
					ActionRequest: &mpi.APIActionRequest{
						InstanceId: nginxInstance.GetInstanceMeta().GetInstanceId(),
						Action:     nil,
					},
				},
			},
			result: true,
		},
		{
			name: "Test 7: invalid action request",
			req: &mpi.ManagementPlaneRequest{
				MessageMeta: protos.CreateMessageMeta(),
				Request: &mpi.ManagementPlaneRequest_ActionRequest{
					ActionRequest: &mpi.APIActionRequest{
						InstanceId: "unknown-id",
						Action:     nil,
					},
				},
			},
			result: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := commandService.isValidRequest(ctx, testCase.req)
			assert.Equal(t, testCase.result, result)
		})
	}
}
