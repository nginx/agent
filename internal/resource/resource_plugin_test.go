// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/nginx/agent/v3/internal/bus/busfakes"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/test/types"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/resource/resourcefakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Process(t *testing.T) {
	ctx := context.Background()

	updatedInstance := &mpi.Instance{
		InstanceConfig: protos.GetNginxOssInstance([]string{}).GetInstanceConfig(),
		InstanceMeta:   protos.GetNginxOssInstance([]string{}).GetInstanceMeta(),
		InstanceRuntime: &mpi.InstanceRuntime{
			ProcessId:  56789,
			BinaryPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetBinaryPath(),
			ConfigPath: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetConfigPath(),
			Details:    protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetDetails(),
		},
	}

	tests := []struct {
		name     string
		message  *bus.Message
		resource *mpi.Resource
		topic    string
	}{
		{
			name: "Test 1: New Instance Topic",
			message: &bus.Message{
				Topic: bus.AddInstancesTopic,
				Data: []*mpi.Instance{
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
				Data: []*mpi.Instance{
					updatedInstance,
				},
			},
			resource: &mpi.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances: []*mpi.Instance{
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
				Data: []*mpi.Instance{
					updatedInstance,
				},
			},
			resource: &mpi.Resource{
				ResourceId: protos.GetHostResource().GetResourceId(),
				Instances:  []*mpi.Instance{},
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
			messagePipe := busfakes.NewFakeMessagePipe()

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

func TestResource_Process_Apply(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		message  *bus.Message
		applyErr error
		topic    []string
	}{
		{
			name: "Test 1: Write Config Successful Topic - Success Status",
			message: &bus.Message{
				Topic: bus.WriteConfigSuccessfulTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         nil,
				},
			},
			applyErr: nil,
			topic:    []string{bus.ConfigApplySuccessfulTopic},
		},
		{
			name: "Test 2: Write Config Successful Topic - Fail Status",
			message: &bus.Message{
				Topic: bus.WriteConfigSuccessfulTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         nil,
				},
			},
			applyErr: errors.New("error reloading"),
			topic:    []string{bus.DataPlaneResponseTopic, bus.ConfigApplyFailedTopic},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.ApplyConfigReturns(test.applyErr)
			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			assert.Equal(t, test.topic[0], messagePipe.GetMessages()[0].Topic)

			if len(test.topic) > 1 {
				assert.Equal(t, test.topic[1], messagePipe.GetMessages()[1].Topic)
			}

			if test.applyErr != nil {
				response, ok := messagePipe.GetMessages()[0].Data.(*mpi.DataPlaneResponse)
				assert.True(tt, ok)
				assert.Equal(tt, test.applyErr.Error(), response.GetCommandResponse().GetError())
			}
		})
	}
}

func TestResource_Process_Rollback(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		message     *bus.Message
		rollbackErr error
		topic       []string
	}{
		{
			name: "Test 1: Rollback Write Topic - Success Status",
			message: &bus.Message{
				Topic: bus.RollbackWriteTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
					InstanceID:    protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         fmt.Errorf("something went wrong with config apply"),
				},
			},
			rollbackErr: nil,
			topic:       []string{bus.ConfigApplyCompleteTopic},
		},
		{
			name: "Test 2: Rollback Write Topic - Fail Status",
			message: &bus.Message{
				Topic: bus.RollbackWriteTopic,
				Data: &model.ConfigApplyMessage{
					CorrelationID: "",
					InstanceID:    protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Error:         fmt.Errorf("something went wrong with config apply"),
				},
			},
			rollbackErr: errors.New("error reloading"),
			topic:       []string{bus.ConfigApplyCompleteTopic, bus.DataPlaneResponseTopic},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeResourceService := &resourcefakes.FakeResourceServiceInterface{}
			fakeResourceService.ApplyConfigReturns(test.rollbackErr)
			messagePipe := busfakes.NewFakeMessagePipe()

			resourcePlugin := NewResource(types.AgentConfig())
			resourcePlugin.resourceService = fakeResourceService

			err := messagePipe.Register(2, []bus.Plugin{resourcePlugin})
			require.NoError(t, err)

			resourcePlugin.messagePipe = messagePipe

			resourcePlugin.Process(ctx, test.message)

			sort.Slice(messagePipe.GetMessages(), func(i, j int) bool {
				return messagePipe.GetMessages()[i].Topic < messagePipe.GetMessages()[j].Topic
			})

			assert.Equal(tt, len(test.topic), len(messagePipe.GetMessages()))

			assert.Equal(t, test.topic[0], messagePipe.GetMessages()[0].Topic)

			if len(test.topic) > 1 {
				assert.Equal(t, test.topic[1], messagePipe.GetMessages()[1].Topic)
			}

			if test.rollbackErr != nil {
				rollbackResponse, ok := messagePipe.GetMessages()[1].Data.(*mpi.DataPlaneResponse)
				assert.True(tt, ok)
				assert.Equal(t, test.topic[1], messagePipe.GetMessages()[1].Topic)
				assert.Equal(tt, test.rollbackErr.Error(), rollbackResponse.GetCommandResponse().GetError())
			}
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
			bus.RollbackWriteTopic,
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

	messagePipe := busfakes.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourcePlugin := NewResource(types.AgentConfig())
	resourcePlugin.resourceService = &resourceService
	err := resourcePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Empty(t, messages)
}
