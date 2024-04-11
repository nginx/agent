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
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceMonitor_Init(t *testing.T) {
	ctx := context.Background()
	resource := &v1.Resource{
		Id:        "123",
		Instances: []*v1.Instance{},
		Info: &v1.Resource_ContainerInfo{
			ContainerInfo: &v1.ContainerInfo{
				ContainerId: "f43f5eg54g54g54",
				Image:       "nginx-agent:v3.0.0",
			},
		},
	}

	mockReourceService := &servicefakes.FakeResourceServiceInterface{}
	mockReourceService.GetResourceReturns(resource)

	messagePipe := bus.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourceMonitor := NewResourceMonitor()
	resourceMonitor.resourceService = mockReourceService
	err := resourceMonitor.Init(ctx, messagePipe)
	require.NoError(t, err)

	messages := messagePipe.GetMessages()

	assert.Len(t, messages, 1)
	assert.Equal(t, bus.ResourceTopic, messages[0].Topic)
	assert.Equal(t, resource, messages[0].Data)
}

func TestResourceMonitor_Info(t *testing.T) {
	resourceMonitor := NewResourceMonitor()
	assert.Equal(t, &bus.Info{Name: "resource-monitor"}, resourceMonitor.Info())
}

func TestResourceMonitor_Subscriptions(t *testing.T) {
	resourceMonitor := NewResourceMonitor()
	assert.Equal(t, []string{bus.InstancesTopic}, resourceMonitor.Subscriptions())
}

func TestResourceMonitor_Process(t *testing.T) {
	ctx := context.Background()
	resource := &v1.Resource{
		Id:        "123",
		Instances: []*v1.Instance{},
		Info: &v1.Resource_ContainerInfo{
			ContainerInfo: &v1.ContainerInfo{
				ContainerId: "f43f5eg54g54g54",
				Image:       "nginx-agent:v3.0.0",
			},
		},
	}

	mockReourceService := &servicefakes.FakeResourceServiceInterface{}
	mockReourceService.GetResourceReturns(resource)

	messagePipe := bus.NewFakeMessagePipe()
	messagePipe.RunWithoutInit(ctx)

	resourceMonitor := NewResourceMonitor()
	resourceMonitor.resourceService = mockReourceService
	err := resourceMonitor.Init(ctx, messagePipe)
	require.NoError(t, err)

	instances := []*v1.Instance{
		{
			InstanceMeta: &v1.InstanceMeta{
				InstanceId:   "123",
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
			},
		},
	}

	resourceMonitor.Process(ctx, &bus.Message{
		Topic: bus.InstancesTopic,
		Data:  instances,
	})

	messages := messagePipe.GetMessages()

	assert.Len(t, messages, 2)
	assert.Equal(t, bus.ResourceTopic, messages[0].Topic)
	assert.Equal(t, resource, messages[0].Data)

	resource.Instances = instances

	assert.Equal(t, bus.ResourceTopic, messages[1].Topic)
	assert.Equal(t, resource, messages[1].Data)
}
