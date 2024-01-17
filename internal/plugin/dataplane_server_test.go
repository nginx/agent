/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/nginx/agent/v3/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestDataplaneServer_Init(t *testing.T) {
	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: &service.InstanceService{},
	})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, dataplaneServer.server.Addr().(*net.TCPAddr).Port)
}

func TestDataplaneServer_Process(t *testing.T) {
	testProcesses := []*os.Process{{Pid: 123, Name: "nginx"}}

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: &service.InstanceService{},
	})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	assert.NoError(t, err)
	go messagePipe.Run()

	messagePipe.Process(&bus.Message{Topic: bus.OS_PROCESSES_TOPIC, Data: testProcesses})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testProcesses, dataplaneServer.processes)
}

func TestDataplaneServer_GetInstances(t *testing.T) {
	expected := &common.Instance{InstanceId: toPtr("ae6c58c1-bc92-30c1-a9c9-85591422068e"), Type: toPtr(common.NGINX), Version: toPtr("1.23.1")}
	instance := &instances.Instance{InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e", Type: instances.Type_NGINX, Version: "1.23.1"}

	instanceService := &service.FakeInstanceServiceInterface{}
	instanceService.GetInstancesReturns([]*instances.Instance{instance}, nil)

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: instanceService,
	})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, dataplaneServer.server.Addr().(*net.TCPAddr).Port)

	target := "http://" + dataplaneServer.server.Addr().(*net.TCPAddr).AddrPort().String() + "/api/v1/instances"
	res, err := http.Get(target)

	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	result := []*common.Instance{}
	err = json.Unmarshal(resBody, &result)
	assert.NoError(t, err)
	assert.Equal(t, expected, result[0])
}
