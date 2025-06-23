// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/model"

	pkg "github.com/nginx/agent/v3/pkg/config"
	"github.com/nginx/agent/v3/pkg/id"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/bus/busfakes"
	"github.com/nginx/agent/v3/internal/config"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/command/commandfakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/stub"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPlugin_Info(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	info := commandPlugin.Info()

	assert.Equal(t, "command", info.Name)
}

func TestCommandPlugin_Subscriptions(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	subscriptions := commandPlugin.Subscriptions()

	assert.Equal(
		t,
		[]string{
			bus.ConnectionResetTopic,
			bus.ResourceUpdateTopic,
			bus.InstanceHealthTopic,
			bus.DataPlaneHealthResponseTopic,
			bus.DataPlaneResponseTopic,
		},
		subscriptions,
	)
}

func TestCommandPlugin_Init(t *testing.T) {
	ctx := context.Background()
	messagePipe := busfakes.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	require.NotNil(t, commandPlugin.messagePipe)
	require.NotNil(t, commandPlugin.commandService)

	commandPlugin.commandService = fakeCommandService

	closeError := commandPlugin.Close(ctx)
	require.NoError(t, closeError)
}

func TestCommandPlugin_createConnection(t *testing.T) {
	ctx := context.Background()
	commandService := &commandfakes.FakeCommandService{}
	commandService.CreateConnectionReturns(&mpi.CreateConnectionResponse{}, nil)
	messagePipe := busfakes.NewFakeMessagePipe()

	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	err := commandPlugin.Init(ctx, messagePipe)
	commandPlugin.commandService = commandService
	require.NoError(t, err)
	defer commandPlugin.Close(ctx)

	commandPlugin.createConnection(ctx, &mpi.Resource{})

	assert.Eventually(
		t,
		func() bool { return commandService.SubscribeCallCount() > 0 },
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.Messages()) == 1 },
		2*time.Second,
		10*time.Millisecond,
	)

	messages := messagePipe.Messages()
	assert.Len(t, messages, 1)
	assert.Equal(t, bus.ConnectionCreatedTopic, messages[0].Topic)
}

func TestCommandPlugin_Process(t *testing.T) {
	ctx := context.Background()
	messagePipe := busfakes.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	defer commandPlugin.Close(ctx)

	// Check CreateConnection
	fakeCommandService.IsConnectedReturnsOnCall(0, false)

	// Check UpdateDataPlaneStatus
	fakeCommandService.IsConnectedReturnsOnCall(1, true)
	fakeCommandService.IsConnectedReturnsOnCall(2, true)

	commandPlugin.commandService = fakeCommandService

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: protos.HostResource()})
	require.Equal(t, 1, fakeCommandService.CreateConnectionCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: protos.HostResource()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneStatusCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.InstanceHealthTopic, Data: protos.InstanceHealths()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneHealthCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: protos.OKDataPlaneResponse()})
	require.Equal(t, 1, fakeCommandService.SendDataPlaneResponseCallCount())

	commandPlugin.Process(ctx, &bus.Message{
		Topic: bus.DataPlaneHealthResponseTopic,
		Data:  protos.HealthyInstanceHealth(),
	})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneHealthCallCount())
	require.Equal(t, 1, fakeCommandService.SendDataPlaneResponseCallCount())

	commandPlugin.Process(ctx, &bus.Message{
		Topic: bus.ConnectionResetTopic,
		Data:  commandPlugin.conn,
	})
	require.Equal(t, 1, fakeCommandService.UpdateClientCallCount())
}

func TestCommandPlugin_monitorSubscribeChannel(t *testing.T) {
	tests := []struct {
		managementPlaneRequest *mpi.ManagementPlaneRequest
		expectedTopic          *bus.Message
		name                   string
		request                string
		configFeatures         []string
	}{
		{
			name: "Test 1: Config Upload Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
					ConfigUploadRequest: &mpi.ConfigUploadRequest{},
				},
			},
			expectedTopic:  &bus.Message{Topic: bus.ConfigUploadRequestTopic},
			request:        "UploadRequest",
			configFeatures: config.DefaultFeatures(),
		},
		{
			name: "Test 2: Config Apply Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: &mpi.ConfigApplyRequest{},
				},
			},
			expectedTopic:  &bus.Message{Topic: bus.ConfigApplyRequestTopic},
			request:        "ApplyRequest",
			configFeatures: config.DefaultFeatures(),
		},
		{
			name: "Test 3: Health Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_HealthRequest{
					HealthRequest: &mpi.HealthRequest{},
				},
			},
			expectedTopic:  &bus.Message{Topic: bus.DataPlaneHealthRequestTopic},
			configFeatures: config.DefaultFeatures(),
		},
		{
			name: "Test 4: API Action Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ActionRequest{
					ActionRequest: &mpi.APIActionRequest{
						Action: &mpi.APIActionRequest_NginxPlusAction{},
					},
				},
			},
			expectedTopic: &bus.Message{Topic: bus.APIActionRequestTopic},
			request:       "APIActionRequest",
			configFeatures: []string{
				pkg.FeatureConfiguration,
				pkg.FeatureMetrics,
				pkg.FeatureFileWatcher,
				pkg.FeatureAPIAction,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			messagePipe := busfakes.NewFakeMessagePipe()

			agentConfig := types.AgentConfig()
			agentConfig.Features = test.configFeatures
			commandPlugin := NewCommandPlugin(agentConfig, &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
			err := commandPlugin.Init(ctx, messagePipe)
			require.NoError(tt, err)
			defer commandPlugin.Close(ctx)

			go commandPlugin.monitorSubscribeChannel(ctx)

			commandPlugin.subscribeChannel <- test.managementPlaneRequest

			assert.Eventually(
				t,
				func() bool { return len(messagePipe.Messages()) == 1 },
				2*time.Second,
				10*time.Millisecond,
			)

			messages := messagePipe.Messages()
			assert.Len(tt, messages, 1)
			assert.Equal(tt, test.expectedTopic.Topic, messages[0].Topic)

			mp, ok := messages[0].Data.(*mpi.ManagementPlaneRequest)

			switch test.request {
			case "UploadRequest":
				assert.True(tt, ok)
				require.NotNil(tt, mp.GetConfigUploadRequest())
			case "ApplyRequest":
				assert.True(tt, ok)
				require.NotNil(tt, mp.GetConfigApplyRequest())
			case "APIActionRequest":
				assert.True(tt, ok)
				require.NotNil(tt, mp.GetActionRequest())
			}
		})
	}
}

func TestCommandPlugin_FeatureDisabled(t *testing.T) {
	tests := []struct {
		managementPlaneRequest *mpi.ManagementPlaneRequest
		expectedLog            string
		name                   string
		request                string
		configFeatures         []string
	}{
		{
			name: "Test 1: Config Upload Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
					ConfigUploadRequest: &mpi.ConfigUploadRequest{},
				},
			},
			expectedLog: "Configuration feature disabled. Unable to process config upload request",
			request:     "UploadRequest",
			configFeatures: []string{
				pkg.FeatureMetrics,
				pkg.FeatureFileWatcher,
			},
		},
		{
			name: "Test 2: Config Apply Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
					ConfigApplyRequest: &mpi.ConfigApplyRequest{},
				},
			},
			expectedLog: "Configuration feature disabled. Unable to process config apply request",
			request:     "ApplyRequest",
			configFeatures: []string{
				pkg.FeatureMetrics,
				pkg.FeatureFileWatcher,
			},
		},
		{
			name: "Test 3: API Action Request",
			managementPlaneRequest: &mpi.ManagementPlaneRequest{
				Request: &mpi.ManagementPlaneRequest_ActionRequest{
					ActionRequest: &mpi.APIActionRequest{
						Action: &mpi.APIActionRequest_NginxPlusAction{},
					},
				},
			},
			expectedLog:    "API Action Request feature disabled. Unable to process API action request",
			request:        "APIActionRequest",
			configFeatures: config.DefaultFeatures(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fakeCommandService := &commandfakes.FakeCommandService{}
			fakeCommandService.SendDataPlaneResponseReturns(nil)
			messagePipe := busfakes.NewFakeMessagePipe()

			agentConfig := types.AgentConfig()

			agentConfig.Features = test.configFeatures

			commandPlugin := NewCommandPlugin(agentConfig, &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
			err := commandPlugin.Init(ctx, messagePipe)
			commandPlugin.commandService = fakeCommandService
			require.NoError(tt, err)
			defer commandPlugin.Close(ctx)

			go commandPlugin.monitorSubscribeChannel(ctx)

			commandPlugin.subscribeChannel <- test.managementPlaneRequest
			assert.Eventually(
				tt,
				func() bool { return fakeCommandService.SendDataPlaneResponseCallCount() == 1 },
				2*time.Second,
				10*time.Millisecond,
			)
		})
	}
}

func TestMonitorSubscribeChannel(t *testing.T) {
	ctx, cncl := context.WithCancel(context.Background())

	logBuf := &bytes.Buffer{}
	stub.StubLoggerWith(logBuf)

	cp := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	cp.subscribeCancel = cncl

	message := protos.CreateManagementPlaneRequest()

	// Run in a separate goroutine
	go cp.monitorSubscribeChannel(ctx)

	// Give some time to exit the goroutine
	time.Sleep(100 * time.Millisecond)

	cp.subscribeChannel <- message

	// Give some time to process the message
	time.Sleep(100 * time.Millisecond)

	cp.Close(ctx)

	time.Sleep(100 * time.Millisecond)

	helpers.ValidateLog(t, "Received management plane request", logBuf)

	// Clear the log buffer
	logBuf.Reset()
}

func Test_createDataPlaneResponse(t *testing.T) {
	expected := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Success",
			Error:   "",
		},
	}
	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{}, model.Command)
	result := commandPlugin.createDataPlaneResponse(expected.GetMessageMeta().GetCorrelationId(),
		expected.GetCommandResponse().GetStatus(),
		expected.GetCommandResponse().GetMessage(), expected.GetCommandResponse().GetError())

	assert.Equal(t, expected.GetCommandResponse(), result.GetCommandResponse())
	assert.Equal(t, expected.GetMessageMeta().GetCorrelationId(), result.GetMessageMeta().GetCorrelationId())
}
