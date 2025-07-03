// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/grpc"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/watcher/credentials"

	"github.com/nginx/agent/v3/internal/bus/busfakes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/watcher/health"
	"github.com/nginx/agent/v3/internal/watcher/instance"
	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/id"
	testModel "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcher_Init(t *testing.T) {
	ctx := context.Background()

	watcherPlugin := NewWatcher(types.AgentConfig())

	messagePipe := busfakes.NewFakeMessagePipe()

	err := watcherPlugin.Init(ctx, messagePipe)
	defer func() {
		closeError := watcherPlugin.Close(ctx)
		require.NoError(t, closeError)
	}()
	require.NoError(t, err)

	messages := messagePipe.Messages()

	assert.Empty(t, messages)

	instanceUpdatesMessage := instance.InstanceUpdatesMessage{
		CorrelationID: logger.GenerateCorrelationID(),
		InstanceUpdates: instance.InstanceUpdates{
			NewInstances: []*mpi.Instance{
				protos.NginxOssInstance([]string{}),
			},
			UpdatedInstances: []*mpi.Instance{
				protos.NginxOssInstance([]string{}),
			},
			DeletedInstances: []*mpi.Instance{
				protos.NginxPlusInstance([]string{}),
			},
		},
	}

	nginxConfigContextMessage := instance.NginxConfigContextMessage{
		CorrelationID:      logger.GenerateCorrelationID(),
		NginxConfigContext: testModel.ConfigContext(),
	}

	instanceHealthMessage := health.InstanceHealthMessage{
		CorrelationID:  logger.GenerateCorrelationID(),
		InstanceHealth: []*mpi.InstanceHealth{},
	}

	credentialUpdateMessage := credentials.CredentialUpdateMessage{
		CorrelationID:  logger.GenerateCorrelationID(),
		ServerType:     model.Command,
		GrpcConnection: &grpc.GrpcConnection{},
	}

	watcherPlugin.instanceUpdatesChannel <- instanceUpdatesMessage
	watcherPlugin.nginxConfigContextChannel <- nginxConfigContextMessage
	watcherPlugin.instanceHealthChannel <- instanceHealthMessage
	watcherPlugin.commandCredentialUpdatesChannel <- credentialUpdateMessage

	assert.Eventually(t, func() bool { return len(messagePipe.Messages()) == 6 }, 2*time.Second, 10*time.Millisecond)
	messages = messagePipe.Messages()

	assert.Equal(
		t,
		&bus.Message{Topic: bus.AddInstancesTopic, Data: instanceUpdatesMessage.InstanceUpdates.NewInstances},
		messages[0],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.UpdatedInstancesTopic, Data: instanceUpdatesMessage.InstanceUpdates.UpdatedInstances},
		messages[1],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.DeletedInstancesTopic, Data: instanceUpdatesMessage.InstanceUpdates.DeletedInstances},
		messages[2],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: nginxConfigContextMessage.NginxConfigContext},
		messages[3],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.InstanceHealthTopic, Data: instanceHealthMessage.InstanceHealth},
		messages[4],
	)
	assert.Equal(t,
		&bus.Message{Topic: bus.ConnectionResetTopic, Data: &grpc.GrpcConnection{}},
		messages[5])
}

func TestWatcher_Info(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(t, &bus.Info{Name: "watcher"}, watcherPlugin.Info())
}

func TestWatcher_Process_CredentialUpdatedTopic(t *testing.T) {
	ctx := context.Background()

	watcherPlugin := NewWatcher(types.AgentConfig())

	messagePipe := busfakes.NewFakeMessagePipe()

	err := watcherPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	message := &bus.Message{
		Topic: bus.CredentialUpdatedTopic,
		Data:  nil,
	}

	watcherPlugin.Process(ctx, message)
}

func TestWatcher_Process_ConfigApplyRequestTopic(t *testing.T) {
	ctx := context.Background()
	data := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: protos.CreateConfigApplyRequest(&mpi.FileOverview{
				ConfigVersion: protos.CreateConfigVersion(),
			}),
		},
	}
	message := &bus.Message{
		Topic: bus.ConfigApplyRequestTopic,
		Data:  data,
	}

	watcherPlugin := NewWatcher(types.AgentConfig())

	watcherPlugin.Process(ctx, message)

	assert.Len(t, watcherPlugin.instancesWithConfigApplyInProgress, 1)
}

func TestWatcher_Process_ConfigApplySuccessfulTopic(t *testing.T) {
	ctx := context.Background()
	data := protos.NginxOssInstance([]string{})

	response := &model.ConfigApplySuccess{
		ConfigContext: &model.NginxConfigContext{
			InstanceID: data.GetInstanceMeta().GetInstanceId(),
		},
		DataPlaneResponse: &mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     id.GenerateMessageID(),
				CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
				Message: "Config apply successful",
				Error:   "",
			},
			InstanceId: data.GetInstanceMeta().GetInstanceId(),
		},
	}

	message := &bus.Message{
		Topic: bus.ConfigApplySuccessfulTopic,
		Data:  response,
	}

	fakeWatcherService := &watcherfakes.FakeInstanceWatcherServiceInterface{}
	watcherPlugin := NewWatcher(types.AgentConfig())
	watcherPlugin.instanceWatcherService = fakeWatcherService
	watcherPlugin.instancesWithConfigApplyInProgress = []string{data.GetInstanceMeta().GetInstanceId()}

	watcherPlugin.Process(ctx, message)

	assert.Equal(t, 1, fakeWatcherService.HandleNginxConfigContextUpdateCallCount())
	assert.Empty(t, watcherPlugin.instancesWithConfigApplyInProgress)
}

func TestWatcher_Process_RollbackCompleteTopic(t *testing.T) {
	ctx := context.Background()
	ossInstance := protos.NginxOssInstance([]string{})

	response := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Config apply successful",
			Error:   "",
		},
		InstanceId: ossInstance.GetInstanceMeta().GetInstanceId(),
	}

	message := &bus.Message{
		Topic: bus.ConfigApplyCompleteTopic,
		Data:  response,
	}

	watcherPlugin := NewWatcher(types.AgentConfig())
	watcherPlugin.instancesWithConfigApplyInProgress = []string{ossInstance.GetInstanceMeta().GetInstanceId()}

	watcherPlugin.Process(ctx, message)

	assert.Empty(t, watcherPlugin.instancesWithConfigApplyInProgress)
}

func TestWatcher_Subscriptions(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(
		t,
		[]string{
			bus.ConfigApplyRequestTopic,
			bus.ConfigApplySuccessfulTopic,
			bus.ConfigApplyCompleteTopic,
			bus.DataPlaneHealthRequestTopic,
		},
		watcherPlugin.Subscriptions(),
	)
}
