// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/resource/resourcefakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Process(t *testing.T) {
	ctx := context.Background()

	updatedInstance := &v1.Instance{
		InstanceConfig: protos.GetNginxOssInstance([]string{}).GetInstanceConfig(),
		InstanceMeta:   protos.GetNginxOssInstance([]string{}).GetInstanceMeta(),
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  56789,
			BinaryPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetBinaryPath(),
			ConfigPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetConfigPath(),
			Details:    protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetDetails(),
		},
	}

	tests := []struct {
		name     string
		message  *bus.Message
		resource *v1.Resource
		topic    string
	}{
		{
			name: "Test 1: New Instance Topic",
			message: &bus.Message{
				Topic: bus.AddInstancesTopic,
				Data: []*v1.Instance{
					protos.GetNginxOssInstance([]string{}),
				},
			},
			resource: protos.GetHostResource(),
			topic:    bus.ResourceUpdateTopic,
		},
		{
			name: "Test 2: Update Instance Topic",
			message: &bus.Message{
				Topic: bus.UpdatedInstancesTopic,
				Data: []*v1.Instance{
					updatedInstance,
				},
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*v1.Instance{
					updatedInstance,
				},
				Info: protos.GetHostResource().GetInfo(),
			},
			topic: bus.ResourceUpdateTopic,
		},
		{
			name: "Test 3: Delete Instance Topic",
			message: &bus.Message{
				Topic: bus.DeletedInstancesTopic,
				Data: []*v1.Instance{
					updatedInstance,
				},
			},
			resource: &v1.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances:  []*v1.Instance{},
				Info:       protos.GetHostResource().GetInfo(),
			},
			topic: bus.ResourceUpdateTopic,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.AddInstancesReturns(protos.GetHostResource())
			fakeResourceService.UpdateInstancesReturns(test.resource)
			fakeResourceService.DeleteInstancesReturns(test.resource)
			messagePipe := bus.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic, messagePipe.GetMessages()[0].Topic)
			assert.Equal(t, test.resource, messagePipe.GetMessages()[0].Data)
		})
	}
}

func TestResource_Subscriptions(t *testing.T) {
	resourcePlugin := NewResource(types.AgentConfig())
	assert.Equal(t,
		[]string{
			bus.AddInstancesTopic,
			bus.UpdatedInstancesTopic,
			bus.DeletedInstancesTopic,
			bus.WriteConfigSuccessfulTopic,
		},
		resourcePlugin.Subscriptions())
}

func TestResource_Info(t *testing.T) {
	resourcePlugin := NewResource(types.AgentConfig())
	assert.Equal(t, &bus.Info{Name: "resource"}, resourcePlugin.Info())
}

func TestResource_Init(t *testing.T) {
	ctx := context.Background()
	resourceService := resourcefakes.FakeResourceServiceInterface{}

	messagePipe := bus.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourcePlugin := NewResource(types.AgentConfig())
	resourcePlugin.resourceService = &resourceService
	err := resourcePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)
}
