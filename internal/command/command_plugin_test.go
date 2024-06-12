// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"testing"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/command/commandfakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPlugin_Info(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	info := commandPlugin.Info()

	assert.Equal(t, "command", info.Name)
}

func TestCommandPlugin_Subscriptions(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	subscriptions := commandPlugin.Subscriptions()

	assert.Equal(
		t,
		[]string{
			bus.ResourceUpdateTopic,
			bus.InstanceHealthTopic,
			bus.DataPlaneResponseTopic,
		},
		subscriptions,
	)
}

func TestCommandPlugin_Init(t *testing.T) {
	ctx := context.Background()
	messagePipe := bus.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	require.NotNil(t, commandPlugin.messagePipe)
	require.NotNil(t, commandPlugin.commandService)

	commandPlugin.commandService = fakeCommandService

	closeError := commandPlugin.Close(ctx)
	require.NoError(t, closeError)
	require.Equal(t, 1, fakeCommandService.CancelSubscriptionCallCount())
}

func TestCommandPlugin_Process(t *testing.T) {
	ctx := context.Background()
	messagePipe := bus.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	defer commandPlugin.Close(ctx)

	commandPlugin.commandService = fakeCommandService

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: protos.GetHostResource()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneStatusCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.InstanceHealthTopic, Data: protos.GetInstanceHealths()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneHealthCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: protos.OKDataPlaneResponse()})
	require.Equal(t, 1, fakeCommandService.SendDataPlaneResponseCallCount())
}

func TestCommandPlugin_monitorSubscribeChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messagePipe := bus.NewFakeMessagePipe()

	commandPlugin := NewCommandPlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	defer commandPlugin.Close(ctx)

	go commandPlugin.monitorSubscribeChannel(ctx)

	commandPlugin.subscribeChannel <- &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{},
		},
	}

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.GetMessages()) == 1 },
		2*time.Second,
		10*time.Millisecond,
	)

	messages := messagePipe.GetMessages()
	assert.Len(t, messages, 1)
	assert.Equal(t, bus.ConfigUploadRequestTopic, messages[0].Topic)

	request, ok := messages[0].Data.(*mpi.ManagementPlaneRequest)
	assert.True(t, ok)
	require.NotNil(t, request.GetConfigUploadRequest())
}
