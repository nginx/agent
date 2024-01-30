/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/datasource"
	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/stretchr/testify/assert"
)

func TestInstanceMonitor_Init(t *testing.T) {
	instanceMonitor := NewInstanceMonitor(&InstanceMonitorParameters{})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{instanceMonitor})
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, instanceMonitor.messagePipe)
}

func TestInstanceMonitor_Info(t *testing.T) {
	instanceMonitor := NewInstanceMonitor(&InstanceMonitorParameters{})
	info := instanceMonitor.Info()
	assert.Equal(t, "instance-monitor", info.Name)
}

func TestInstanceMonitor_Subscriptions(t *testing.T) {
	instanceMonitor := NewInstanceMonitor(&InstanceMonitorParameters{})
	subscriptions := instanceMonitor.Subscriptions()
	assert.Equal(t, []string{bus.OS_PROCESSES_TOPIC}, subscriptions)
}

func TestInstanceMonitor_Process(t *testing.T) {
	testInstances := []*instances.Instance{{InstanceId: "123", Type: instances.Type_NGINX}}

	fakeNginxDatasource := &datasource.FakeDatasource{}
	fakeNginxDatasource.GetInstancesReturns(testInstances, nil)
	instanceMonitor := NewInstanceMonitor(&InstanceMonitorParameters{nginxDatasource: fakeNginxDatasource})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{instanceMonitor})
	assert.NoError(t, err)
	go messagePipe.Run()

	messagePipe.Process(&bus.Message{Topic: bus.OS_PROCESSES_TOPIC, Data: []*os.Process{{Pid: 123, Name: "nginx"}}})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testInstances, instanceMonitor.instances)
}
