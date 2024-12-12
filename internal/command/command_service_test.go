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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	time.Sleep(1 * time.Second)

	return nil, nil
}

type FakeConfigApplySubscribeClient struct {
	grpc.ClientStream
}

func (*FakeConfigApplySubscribeClient) Send(*mpi.DataPlaneResponse) error {
	return nil
}

// nolint: nilnil
func (*FakeConfigApplySubscribeClient) Recv() (*mpi.ManagementPlaneRequest, error) {
	return &mpi.ManagementPlaneRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     "1",
			CorrelationId: "123",
			Timestamp:     timestamppb.Now(),
		},
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: &mpi.ConfigApplyRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "12314",
						Version:    "4215432",
					},
				},
			},
		},
	}, nil
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

func TestCommandService_receiveCallback_configApplyRequest(t *testing.T) {
	ctx := context.Background()
	fakeSubscribeClient := &FakeConfigApplySubscribeClient{}

	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	commandServiceClient.SubscribeReturns(fakeSubscribeClient, nil)

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

	assert.Len(t, commandService.configApplyRequestQueue, 1)
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

	if s := logBuf.String(); !strings.Contains(s, "Failed to send update data plane status") {
		t.Errorf("Unexpected log %s", s)
	}
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

func TestCommandService_SendDataPlaneResponse_configApplyRequest(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	subscribeClient := &FakeSubscribeClient{}
	subscribeChannel := make(chan *mpi.ManagementPlaneRequest)

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.AgentConfig(),
		subscribeChannel,
	)

	request1 := &mpi.ManagementPlaneRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     "1",
			CorrelationId: "123",
			Timestamp:     timestamppb.Now(),
		},
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: &mpi.ConfigApplyRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "12314",
						Version:    "4215432",
					},
				},
			},
		},
	}

	request2 := &mpi.ManagementPlaneRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     "2",
			CorrelationId: "1232",
			Timestamp:     timestamppb.Now(),
		},
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: &mpi.ConfigApplyRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "12314",
						Version:    "4215432",
					},
				},
			},
		},
	}

	request3 := &mpi.ManagementPlaneRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     "3",
			CorrelationId: "1233",
			Timestamp:     timestamppb.Now(),
		},
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: &mpi.ConfigApplyRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "12314",
						Version:    "4215432",
					},
				},
			},
		},
	}

	commandService.configApplyRequestQueueMutex.Lock()
	commandService.configApplyRequestQueue = map[string][]*mpi.ManagementPlaneRequest{
		"12314": {
			request1,
			request2,
			request3,
		},
	}
	commandService.configApplyRequestQueueMutex.Unlock()

	commandService.subscribeClientMutex.Lock()
	commandService.subscribeClient = subscribeClient
	commandService.subscribeClientMutex.Unlock()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		requestFromChannel := <-subscribeChannel
		assert.Equal(t, request3, requestFromChannel)
		wg.Done()
	}()

	err := commandService.SendDataPlaneResponse(
		ctx,
		&mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: "1232",
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
				Message: "Success",
			},
			InstanceId: "12314",
		},
	)

	require.NoError(t, err)

	assert.Len(t, commandService.configApplyRequestQueue, 1)
	assert.Equal(t, request3, commandService.configApplyRequestQueue["12314"][0])
	wg.Wait()
}
