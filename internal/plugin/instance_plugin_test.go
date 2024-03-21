// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
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
	testInstances := []*instances.Instance{{InstanceId: "123", Type: instances.Type_NGINX}}

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testInstances)
	instanceMonitor := NewInstance()
	instanceMonitor.instanceService = fakeInstanceService

	messagePipe := bus.NewMessagePipe(100)
	err := messagePipe.Register(100, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)
	go messagePipe.Run(ctx)

	messagePipe.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{{Pid: 123, Name: "nginx"}}})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testInstances, instanceMonitor.getInstances())
}

func TestInstance_Process_Error_Expected(t *testing.T) {
	ctx := context.Background()
	instanceMonitor := NewInstance()

	messagePipe := bus.NewMessagePipe(2)
	err := messagePipe.Register(2, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)
	go messagePipe.Run(ctx)

	messagePipe.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: nil})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, []*instances.Instance{}, instanceMonitor.instances)
}

func TestInstance_Process_Empty_Instances(t *testing.T) {
	ctx := context.Background()
	testInstances := []*instances.Instance{}

	fakeInstanceService := &servicefakes.FakeInstanceServiceInterface{}
	fakeInstanceService.GetInstancesReturns(testInstances)
	instanceMonitor := NewInstance()

	messagePipe := bus.NewMessagePipe(2)
	err := messagePipe.Register(2, []bus.Plugin{instanceMonitor})
	require.NoError(t, err)
	go messagePipe.Run(ctx)

	messagePipe.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: []*model.Process{}})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testInstances, instanceMonitor.instances)
}
