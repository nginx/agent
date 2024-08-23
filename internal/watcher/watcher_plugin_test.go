// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/watcher/health"
	"github.com/nginx/agent/v3/internal/watcher/instance"
	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcher_Init(t *testing.T) {
	ctx := context.Background()

	watcherPlugin := NewWatcher(types.AgentConfig())

	messagePipe := bus.NewFakeMessagePipe()

	err := watcherPlugin.Init(ctx, messagePipe)
	defer func() {
		closeError := watcherPlugin.Close(ctx)
		require.NoError(t, closeError)
	}()
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)

	instanceUpdatesMessage := instance.InstanceUpdatesMessage{
		CorrelationID: logger.GenerateCorrelationID(),
		InstanceUpdates: instance.InstanceUpdates{
			NewInstances: []*mpi.Instance{
				protos.GetNginxOssInstance([]string{}),
			},
			UpdatedInstances: []*mpi.Instance{
				protos.GetNginxOssInstance([]string{}),
			},
			DeletedInstances: []*mpi.Instance{
				protos.GetNginxPlusInstance([]string{}),
			},
		},
	}

	nginxConfigContextMessage := instance.NginxConfigContextMessage{
		CorrelationID:      logger.GenerateCorrelationID(),
		NginxConfigContext: model.GetConfigContext(),
	}

	instanceHealthMessage := health.InstanceHealthMessage{
		CorrelationID:  logger.GenerateCorrelationID(),
		InstanceHealth: []*mpi.InstanceHealth{},
	}

	watcherPlugin.instanceUpdatesChannel <- instanceUpdatesMessage
	watcherPlugin.nginxConfigContextChannel <- nginxConfigContextMessage
	watcherPlugin.instanceHealthChannel <- instanceHealthMessage

	assert.Eventually(t, func() bool { return len(messagePipe.GetMessages()) == 5 }, 2*time.Second, 10*time.Millisecond)
	messages = messagePipe.GetMessages()

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
}

func TestWatcher_Info(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(t, &bus.Info{Name: "watcher"}, watcherPlugin.Info())
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
	data := protos.GetNginxOssInstance([]string{})
	message := &bus.Message{
		Topic: bus.ConfigApplySuccessfulTopic,
		Data:  data,
	}

	fakeWatcherService := &watcherfakes.FakeInstanceWatcherServiceInterface{}
	watcherPlugin := NewWatcher(types.AgentConfig())
	watcherPlugin.instanceWatcherService = fakeWatcherService
	watcherPlugin.instancesWithConfigApplyInProgress = []string{data.GetInstanceMeta().GetInstanceId()}

	watcherPlugin.Process(ctx, message)

	assert.Equal(t, 1, fakeWatcherService.ReparseConfigCallCount())
	assert.Empty(t, watcherPlugin.instancesWithConfigApplyInProgress)
}

func TestWatcher_Process_RollbackCompleteTopic(t *testing.T) {
	ctx := context.Background()
	instanceID := "123"
	message := &bus.Message{
		Topic: bus.RollbackCompleteTopic,
		Data:  instanceID,
	}

	watcherPlugin := NewWatcher(types.AgentConfig())
	watcherPlugin.instancesWithConfigApplyInProgress = []string{instanceID}

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
			bus.RollbackCompleteTopic,
		},
		watcherPlugin.Subscriptions(),
	)
}
