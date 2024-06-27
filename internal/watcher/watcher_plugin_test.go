// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"github.com/nginx/agent/v3/internal/watcher/instance"
	"github.com/nginx/agent/v3/internal/watcher/watcherfakes"
	"testing"
	"time"

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

	watcherPlugin.instanceUpdatesChannel <- instanceUpdatesMessage
	watcherPlugin.nginxConfigContextChannel <- nginxConfigContextMessage

	assert.Eventually(t, func() bool { return len(messagePipe.GetMessages()) == 4 }, 2*time.Second, 10*time.Millisecond)
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
}

func TestWatcher_Info(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(t, &bus.Info{Name: "watcher"}, watcherPlugin.Info())
}

func TestWatcher_Process(t *testing.T) {
	ctx := context.Background()
	message := &bus.Message{
		Topic: bus.ConfigApplySuccessfulTopic,
		Data:  protos.GetNginxOssInstance([]string{}),
	}

	fakeWatcherService := &watcherfakes.FakeInstanceWatcherServiceInterface{}
	watcherPlugin := NewWatcher(types.AgentConfig())
	watcherPlugin.instanceWatcherService = fakeWatcherService

	watcherPlugin.Process(ctx, message)

	require.Equal(t, 1, fakeWatcherService.ReparseConfigCallCount())
}

func TestWatcher_Subscriptions(t *testing.T) {
	watcherPlugin := NewWatcher(types.AgentConfig())
	assert.Equal(t, []string{bus.ConfigApplySuccessfulTopic}, watcherPlugin.Subscriptions())
}
