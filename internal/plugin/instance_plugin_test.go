// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
)

func TestInstance_Info(t *testing.T) {
	instanceMonitor := NewInstance()
	info := instanceMonitor.Info()
	assert.Equal(t, "instance", info.Name)
}

func TestInstance_Subscriptions(t *testing.T) {
	instanceMonitor := NewInstance()
	subscriptions := instanceMonitor.Subscriptions()
	assert.Equal(t, []string{bus.OsProcessesTopic, bus.InstanceConfigUpdateRequestTopic}, subscriptions)
}

func TestInstance_Process(t *testing.T) {
	ctx := context.Background()
	testInstances := []*v1.Instance{
		{
			InstanceMeta: &v1.InstanceMeta{
				InstanceId:   "123",
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
			},
		},
	}

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testInstances)
	instanceMonitor := NewInstance()
	instanceMonitor.instanceService = fakeInstanceService

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(100, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)

	processesMessage := &bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{{Pid: 123, Name: "nginx"}}}
	messagePipe.Process(ctx, processesMessage)
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 2)
	assert.Equal(t, processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Equal(t, processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
	assert.Equal(t, bus.InstancesTopic, messagePipe.GetProcessedMessages()[1].Topic)
	assert.Equal(t, testInstances, messagePipe.GetProcessedMessages()[1].Data)
}

func TestInstance_Process_Error_Expected(t *testing.T) {
	ctx := context.Background()
	instanceMonitor := NewInstance()

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)

	messagePipe.Process(ctx, &bus.Message{Topic: bus.OsProcessesTopic, Data: nil})
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 1)
	assert.Equal(t, bus.OsProcessesTopic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Nil(t, messagePipe.GetProcessedMessages()[0].Data)
}

func TestInstance_Process_Empty_Instances(t *testing.T) {
	ctx := context.Background()
	testInstances := []*v1.Instance{}

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testInstances)
	instanceMonitor := NewInstance()

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)

	processesMessage := &bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{}}
	messagePipe.Process(ctx, processesMessage)
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 2)
	assert.Equal(t, processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Equal(t, processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
	assert.Equal(t, bus.InstancesTopic, messagePipe.GetProcessedMessages()[1].Topic)
	assert.Equal(t, testInstances, messagePipe.GetProcessedMessages()[1].Data)
}
