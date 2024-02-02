// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/service"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
)

func TestDataplaneServer_Init(t *testing.T) {
	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: service.NewInstanceService(),
	})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataplaneServer.server.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	assert.NotNil(t, addr.Port)
}

func TestDataplaneServer_Process(t *testing.T) {
	testInstances := []*instances.Instance{{InstanceId: "123", Type: instances.Type_NGINX}}

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: service.NewInstanceService(),
	})

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	messagePipe.Process(&bus.Message{Topic: bus.InstancesTopic, Data: testInstances})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testInstances, dataplaneServer.instances)
}

func TestDataplaneServer_GetInstances(t *testing.T) {
	ctx := context.TODO()
	expected := &common.Instance{
		InstanceId: toPtr("ae6c58c1-bc92-30c1-a9c9-85591422068e"),
		Type:       toPtr(common.NGINX),
		Version:    toPtr("1.23.1"),
	}

	instance := &instances.Instance{
		InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e",
		Type:       instances.Type_NGINX,
		Version:    "1.23.1",
	}

	instanceService := &servicefakes.FakeInstanceServiceInterface{}

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: instanceService,
	})

	dataplaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(ctx, 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataplaneServer.server.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	assert.NotNil(t, addr.Port)

	target := "http://" + addr.AddrPort().String() + "/api/v1/instances"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	err = res.Body.Close()
	require.NoError(t, err)

	result := []*common.Instance{}
	err = json.Unmarshal(resBody, &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, expected, result[0])
}

func TestDataplaneServer_UpdateInstanceConfiguration(t *testing.T) {
	ctx := context.TODO()
	unknownInstanceID := "fe4c58c1-bc92-30c1-a9c9-85591422068e"
	instanceID := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	data := []byte(`{"location": "http://file-server.com"}`)
	instance := &instances.Instance{InstanceId: instanceID, Type: instances.Type_NGINX, Version: "1.23.1"}

	instanceService := &servicefakes.FakeInstanceServiceInterface{}
	instanceService.GetInstancesReturns([]*instances.Instance{instance})

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: instanceService,
	})

	dataplaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataplaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataplaneServer.server.Addr().(*net.TCPAddr)
	assert.True(t, ok)

	assert.NotNil(t, addr.Port)

	tests := []struct {
		name               string
		instanceID         string
		expectedStatusCode int
		expectedMessage    string
	}{
		{
			name:               "Update known instance configuration",
			instanceID:         instanceID,
			expectedStatusCode: 200,
		},
		{
			name:               "Update unknown instance configuration",
			instanceID:         unknownInstanceID,
			expectedStatusCode: 404,
			expectedMessage:    fmt.Sprintf("Unable to find instance %s", unknownInstanceID),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			res, err := performPutRequest(ctx, dataplaneServer, test.instanceID, data)
			require.NoError(tt, err)
			assert.Equal(tt, test.expectedStatusCode, res.StatusCode)

			resBody, err := io.ReadAll(res.Body)
			require.NoError(tt, err)

			err = res.Body.Close()
			require.NoError(t, err)

			if test.expectedMessage == "" {
				result := dataplane.CorrelationId{}
				err = json.Unmarshal(resBody, &result)
				require.NoError(tt, err)
				assert.NotEmpty(tt, result.CorrelationId)
			} else {
				result := common.ErrorResponse{}
				err = json.Unmarshal(resBody, &result)
				require.NoError(tt, err)
				assert.Equal(tt, test.expectedMessage, result.Message)
			}
		})
	}
}

func performPutRequest(
	ctx context.Context,
	dataplaneServer *DataplaneServer,
	instanceID string,
	data []byte,
) (*http.Response, error) {
	addr, ok := dataplaneServer.server.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("unable to get server address")
	}
	target := "http://" + addr.AddrPort().String() + "/api/v1/instances/" + instanceID + "/configurations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewBuffer(data))
	if err != nil {
		return &http.Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	return client.Do(req)
}
