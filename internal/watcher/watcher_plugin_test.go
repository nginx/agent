// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatcher_Init(t *testing.T) {
	ctx := context.Background()

	watcherPlugin := NewWatcher(types.GetAgentConfig())

	messagePipe := bus.NewFakeMessagePipe()

	err := watcherPlugin.Init(ctx, messagePipe)
	defer func() {
		closeError := watcherPlugin.Close(ctx)
		require.NoError(t, closeError)
	}()
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)

	instanceUpdates := InstanceUpdates{
		newInstances: []*v1.Instance{
			protos.GetNginxOssInstance([]string{}),
		},
		deletedInstances: []*v1.Instance{
			protos.GetNginxPlusInstance([]string{}),
		},
	}

	watcherPlugin.instanceUpdatesChannel <- instanceUpdates

	messages = messagePipe.GetMessages()

	assert.Eventually(t, func() bool { return len(messages) == 2 }, 1*time.Second, 10*time.Millisecond)
	assert.Equal(t, &bus.Message{Topic: bus.NewInstancesTopic, Data: instanceUpdates.newInstances}, messages[0])
	assert.Equal(t, &bus.Message{Topic: bus.DeletedInstancesTopic, Data: instanceUpdates.deletedInstances}, messages[1])
}

func TestWatcher_Info(t *testing.T) {
	watcherPlugin := NewWatcher(types.GetAgentConfig())
	assert.Equal(t, &bus.Info{Name: "watcher"}, watcherPlugin.Info())
}

func TestWatcher_Subscriptions(t *testing.T) {
	watcherPlugin := NewWatcher(types.GetAgentConfig())
	assert.Equal(t, []string{}, watcherPlugin.Subscriptions())
}
