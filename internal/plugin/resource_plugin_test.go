// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
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

	resourcePlugin := NewResource(types.GetAgentConfig())
	resourcePlugin.resourceService = mockReourceService
	err := resourcePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)
}

func TestResourceMonitor_Info(t *testing.T) {
	resourcePlugin := NewResource(types.GetAgentConfig())
	assert.Equal(t, &bus.Info{Name: "resource"}, resourcePlugin.Info())
}

func TestResourceMonitor_Subscriptions(t *testing.T) {
	resourcePlugin := NewResource(types.GetAgentConfig())
	assert.Equal(t,
		[]string{
			bus.OsProcessesTopic,
			bus.InstanceConfigContextTopic,
		},
		resourcePlugin.Subscriptions())
}

func TestResource_Instances_Process(t *testing.T) {
	ctx := context.Background()

	nginxConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		StubStatus: "http://127.0.0.1:8081/api",
		InstanceID: protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
	}

	tests := []struct {
		name             string
		processesMessage *bus.Message
		resource         *v1.Resource
		topic            string
	}{
		{
			name: "Test 1: OS Process Topic",
			processesMessage: &bus.Message{
				Topic: bus.OsProcessesTopic,
				Data:  map[int32]*model.Process{123: {Pid: 123, Name: "nginx"}},
			},
			resource: protos.GetHostResource(),
			topic:    bus.ResourceTopic,
		},
		{
			name:             "Test 2: Instance Config Context Topic",
			processesMessage: &bus.Message{Topic: bus.InstanceConfigContextTopic, Data: nginxConfigContext},
			resource:         protos.GetHostResource(),
			topic:            bus.ResourceTopic,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeResourceService := &servicefakes.FakeResourceServiceInterface{}
			fakeResourceService.GetResourceReturns(test.resource)

			fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
			fakeInstanceService.GetInstancesReturns(test.resource.GetInstances())

			resourcePlugin := NewResource(types.GetAgentConfig())
			resourcePlugin.instanceService = fakeInstanceService
			resourcePlugin.resourceService = fakeResourceService

			messagePipe := bus.NewFakeMessagePipe()
			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			messagePipe.Process(ctx, test.processesMessage)
			messagePipe.Run(ctx)

			slog.Info("messages", "", messagePipe.GetProcessedMessages()[0].Data)
			assert.Len(t, messagePipe.GetProcessedMessages(), 2)
			assert.Equal(t, test.processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
			assert.Equal(t, test.processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
			assert.Equal(t, test.topic, messagePipe.GetProcessedMessages()[1].Topic)
			assert.Equal(t, test.resource, messagePipe.GetProcessedMessages()[1].Data)
		})
	}
}

func TestResource_Process_Error_Expected(t *testing.T) {
	ctx := context.Background()
	testResource := protos.GetHostResource()

	fakeResourceService := &servicefakes.FakeResourceServiceInterface{}
	fakeResourceService.GetResourceReturns(testResource)

	resourcePlugin := NewResource(types.GetAgentConfig())
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

	resourcePlugin := NewResource(types.GetAgentConfig())
	resourcePlugin.instanceService = fakeInstanceService
	resourcePlugin.resourceService = fakeResourceService

	messagePipe := bus.NewFakeMessagePipe()
	err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
	require.NoError(t, err)

	processesMessage := &bus.Message{Topic: bus.OsProcessesTopic, Data: make(map[int32]*model.Process)}
	messagePipe.Process(ctx, processesMessage)
	messagePipe.Run(ctx)

	assert.Len(t, messagePipe.GetProcessedMessages(), 2)
	assert.Equal(t, processesMessage.Topic, messagePipe.GetProcessedMessages()[0].Topic)
	assert.Equal(t, processesMessage.Data, messagePipe.GetProcessedMessages()[0].Data)
	assert.Equal(t, bus.ResourceTopic, messagePipe.GetProcessedMessages()[1].Topic)
	assert.Equal(t, testResource, messagePipe.GetProcessedMessages()[1].Data)
}

func TestResource_Instances_updateInstance(t *testing.T) {
	resourcePlugin := NewResource(types.GetAgentConfig())

	nginxOSSConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		StubStatus: "http://127.0.0.1:8081/api",
	}

	nginxPlusConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name: "/usr/local/var/log/nginx/access.log",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name: "/usr/local/var/log/nginx/error.log",
			},
		},
		PlusAPI: "http://127.0.0.1:8081/api",
	}

	tests := []struct {
		name               string
		nginxConfigContext *model.NginxConfigContext
		instance           *v1.Instance
	}{
		{
			name:               "Test 1: OSS Instance",
			nginxConfigContext: nginxOSSConfigContext,
			instance:           protos.GetNginxOssInstance([]string{}),
		},
		{
			name:               "Test 2: Plus Instance",
			nginxConfigContext: nginxPlusConfigContext,
			instance:           protos.GetNginxPlusInstance([]string{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resourcePlugin.updateInstance(test.nginxConfigContext, test.instance)
			if test.name == "Test 2: Plus Instance" {
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetStubStatus())
				assert.Equal(t, test.nginxConfigContext.PlusAPI, test.instance.GetInstanceRuntime().
					GetNginxPlusRuntimeInfo().GetPlusApi())
			} else {
				assert.Equal(t, test.nginxConfigContext.AccessLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetAccessLogs()[0])
				assert.Equal(t, test.nginxConfigContext.ErrorLogs[0].Name, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetErrorLogs()[0])
				assert.Equal(t, test.nginxConfigContext.StubStatus, test.instance.GetInstanceRuntime().
					GetNginxRuntimeInfo().GetStubStatus())
			}
		})
	}
}
