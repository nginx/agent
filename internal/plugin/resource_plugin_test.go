// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Init(t *testing.T) {
	ctx := context.Background()
	resource := protos.GetContainerizedResource()

	mockReourceService := &servicefakes.FakeResourceServiceInterface{}
	mockReourceService.GetResourceReturns(resource)

	messagePipe := bus.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourcePlugin := NewResource()
	resourcePlugin.resourceService = mockReourceService
	err := resourcePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)
}

func TestResourceMonitor_Info(t *testing.T) {
	resource := NewResource()
	assert.Equal(t, &bus.Info{Name: "resource"}, resource.Info())
}

func TestResourceMonitor_Subscriptions(t *testing.T) {
	resource := NewResource()
	assert.Equal(t,
		[]string{
			bus.OsProcessesTopic,
		},
		resource.Subscriptions())
}

func TestResource_Instances_Process(t *testing.T) {
	ctx := context.Background()
	testResource := protos.GetHostResource()

	fakeResourceService := &servicefakes.FakeResourceServiceInterface{}
	fakeResourceService.GetResourceReturns(testResource)

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testResource.GetInstances())

	resourcePlugin := NewResource()
	resourcePlugin.instanceService = fakeInstanceService
	resourcePlugin.resourceService = fakeResourceService

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
	require.NoError(t, err)

	processesMessage := &bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{{Pid: 123, Name: "nginx"}}}
	messagePipe.Process(ctx, processesMessage)
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 2)
	assert.Equal(t, processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Equal(t, processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
	assert.Equal(t, bus.ResourceTopic, messagePipe.GetProcessedMessages()[1].Topic)
	assert.Equal(t, testResource, messagePipe.GetProcessedMessages()[1].Data)
}

func TestResource_Process_Error_Expected(t *testing.T) {
	ctx := context.Background()
	testResource := protos.GetHostResource()

	fakeResourceService := &servicefakes.FakeResourceServiceInterface{}
	fakeResourceService.GetResourceReturns(testResource)

	resourcePlugin := NewResource()
	resourcePlugin.resourceService = fakeResourceService

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
	require.NoError(t, err)

	messagePipe.Process(ctx, &bus.Message{Topic: bus.OsProcessesTopic, Data: nil})
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 1)
	assert.Equal(t, bus.OsProcessesTopic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Nil(t, messagePipe.GetProcessedMessages()[0].Data)
}

func TestResource_Process_Empty_Instances(t *testing.T) {
	ctx := context.Background()
	testInstances := []*v1.Instance{}
	testResource := protos.GetHostResource()

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testInstances)

	fakeResourceService := &servicefakes.FakeResourceServiceInterface{}
	fakeResourceService.GetResourceReturns(testResource)

	resource := NewResource()
	resource.instanceService = fakeInstanceService
	resource.resourceService = fakeResourceService

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{resource})
	require.NoError(t, err)

	processesMessage := &bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{}}
	messagePipe.Process(ctx, processesMessage)
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 2)
	assert.Equal(t, processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Equal(t, processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
	assert.Equal(t, bus.ResourceTopic, messagePipe.GetProcessedMessages()[1].Topic)
	assert.Equal(t, testResource, messagePipe.GetProcessedMessages()[1].Data)
}
