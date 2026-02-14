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
	"github.com/nginx/agent/v3/internal/watcher/health"
	"github.com/nginx/agent/v3/internal/watcher/instance"
	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/logger"
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

	resourceUpdatesMessage := instance.ResourceUpdatesMessage{
		CorrelationID: logger.GenerateCorrelationID(),
		Resource: &mpi.Resource{
			ResourceId: protos.HostResource().GetResourceId(),
			Instances: []*mpi.Instance{
				protos.NginxOssInstance([]string{}),
			},
			Info: protos.HostResource().GetInfo(),
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

	watcherPlugin.resourceUpdatesChannel <- resourceUpdatesMessage
	watcherPlugin.nginxConfigContextChannel <- nginxConfigContextMessage
	watcherPlugin.instanceHealthChannel <- instanceHealthMessage
	watcherPlugin.commandCredentialUpdatesChannel <- credentialUpdateMessage

	assert.Eventually(t, func() bool { return len(messagePipe.Messages()) == 4 }, 2*time.Second, 10*time.Millisecond)
	messages = messagePipe.Messages()

	assert.Equal(
		t,
		&bus.Message{Topic: bus.ResourceUpdateTopic, Data: resourceUpdatesMessage.Resource},
		messages[0],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: nginxConfigContextMessage.NginxConfigContext},
		messages[1],
	)
	assert.Equal(
		t,
		&bus.Message{Topic: bus.InstanceHealthTopic, Data: instanceHealthMessage.InstanceHealth},
		messages[2],
	)
	assert.Equal(t,
		&bus.Message{Topic: bus.ConnectionResetTopic, Data: &grpc.GrpcConnection{}},
		messages[3])
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

	tests := []struct {
		data       *model.EnableWatchers
		name       string
		inProgress []string
		callCount  int
		empty      bool
	}{
		{
			name: "Test 1: Reparse Config",
			data: &model.EnableWatchers{
				ConfigContext: &model.NginxConfigContext{
					InstanceID: data.GetInstanceMeta().GetInstanceId(),
				},
				InstanceID: data.GetInstanceMeta().GetInstanceId(),
			},
			callCount: 1,
			empty:     true,
			inProgress: []string{
				data.GetInstanceMeta().GetInstanceId(),
			},
		},
		{
			name: "Test 2: Don't Reparse Config",
			data: &model.EnableWatchers{
				ConfigContext: &model.NginxConfigContext{},
				InstanceID:    data.GetInstanceMeta().GetInstanceId(),
			},
			callCount: 0,
			empty:     true,
			inProgress: []string{
				data.GetInstanceMeta().GetInstanceId(),
			},
		},
		{
			name: "Test 3: More than one inProgress Config",
			data: &model.EnableWatchers{
				ConfigContext: &model.NginxConfigContext{
					InstanceID: data.GetInstanceMeta().GetInstanceId(),
				},
				InstanceID: data.GetInstanceMeta().GetInstanceId(),
			},
			callCount: 1,
			empty:     false,
			inProgress: []string{
				data.GetInstanceMeta().GetInstanceId(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := &bus.Message{
				Topic: bus.EnableWatchersTopic,
				Data:  test.data,
			}

			fakeWatcherService := &watcherfakes.FakeInstanceWatcherServiceInterface{}
			watcherPlugin := NewWatcher(types.AgentConfig())
			watcherPlugin.instanceWatcherService = fakeWatcherService
			watcherPlugin.instancesWithConfigApplyInProgress = test.inProgress

			watcherPlugin.Process(ctx, message)

			assert.Equal(t, test.callCount, fakeWatcherService.HandleNginxConfigContextUpdateCallCount())
			if test.empty {
				assert.Empty(t, watcherPlugin.instancesWithConfigApplyInProgress)
			} else {
				assert.NotEmpty(t, watcherPlugin.instancesWithConfigApplyInProgress)
			}
		})
	}
}

func TestWatcher_Subscriptions(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(
		t,
		[]string{
			bus.ConfigApplyRequestTopic,
			bus.DataPlaneHealthRequestTopic,
			bus.EnableWatchersTopic,
			bus.AgentConfigUpdateTopic,
		},
		watcherPlugin.Subscriptions(),
	)
}
