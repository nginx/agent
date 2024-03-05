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
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDataPlaneServer_Init(t *testing.T) {
	dataPlaneServer := NewDataPlaneServer(&config.Config{}, slog.Default())

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	assert.NotNil(t, addr.Port)
}

func TestDataPlaneServer_Process(t *testing.T) {
	dataPlaneServer := NewDataPlaneServer(&config.Config{}, slog.Default())

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	tests := []struct {
		name  string
		data  interface{}
		topic string
	}{
		{
			name:  "instances test",
			data:  []*instances.Instance{{InstanceId: "123", Type: instances.Type_NGINX}},
			topic: bus.InstancesTopic,
		},
		{
			name: "instances complete update",
			data: &instances.ConfigurationStatus{
				InstanceId:    "123",
				CorrelationId: "456",
				Status:        instances.Status_SUCCESS,
				Message:       "config updated",
			},
			topic: bus.InstanceConfigUpdateTopic,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messagePipe.Process(&bus.Message{Topic: tt.topic, Data: tt.data})
			time.Sleep(10 * time.Millisecond)

			var actual interface{}
			if tt.topic == bus.InstancesTopic {
				actual = dataPlaneServer.instances
			} else {
				actual = dataPlaneServer.configEvents["123"][0]
			}

			assert.Equal(t, tt.data, actual)
		})
	}
}

func TestDataPlaneServer_GetInstances(t *testing.T) {
	ctx := context.TODO()
	expected := &dataplane.Instance{
		InstanceId: toPtr("ae6c58c1-bc92-30c1-a9c9-85591422068e"),
		Type:       toPtr(dataplane.NGINX),
		Version:    toPtr("1.23.1"),
	}

	instance := &instances.Instance{
		InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e",
		Type:       instances.Type_NGINX,
		Version:    "1.23.1",
	}

	dataPlaneServer := NewDataPlaneServer(&config.Config{}, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(ctx, 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)
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

	result := []*dataplane.Instance{}
	err = json.Unmarshal(resBody, &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, expected, result[0])
}

func TestDataPlaneServer_UpdateInstanceConfiguration(t *testing.T) {
	ctx := context.TODO()
	unknownInstanceID := "fe4c58c1-bc92-30c1-a9c9-85591422068e"
	instanceID := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	data := []byte(`{"location": "http://file-server.com"}`)
	instance := &instances.Instance{InstanceId: instanceID, Type: instances.Type_NGINX, Version: "1.23.1"}

	dataPlaneServer := NewDataPlaneServer(&config.Config{}, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)
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
			res, err := performPutRequest(ctx, dataPlaneServer, test.instanceID, data)
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
				result := dataplane.ErrorResponse{}
				err = json.Unmarshal(resBody, &result)
				require.NoError(tt, err)
				assert.Equal(tt, test.expectedMessage, result.Message)
			}
		})
	}
}

func TestDataPlaneServer_GetInstanceConfigurationStatus(t *testing.T) {
	ctx := context.TODO()
	correlationID := "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	instanceID := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	tests := []struct {
		name             string
		instanceID       string
		events           []*instances.ConfigurationStatus
		expectedStatus   *dataplane.ConfigurationStatus
		expectedResponse int
	}{
		{
			name:       "happy path",
			instanceID: instanceID,
			events: []*instances.ConfigurationStatus{
				{
					InstanceId:    instanceID,
					CorrelationId: correlationID,
					Status:        instances.Status_SUCCESS,
					Message:       "Success",
				},
				{
					InstanceId:    instanceID,
					CorrelationId: correlationID,
					Status:        instances.Status_IN_PROGRESS,
					Message:       "In Progress",
				},
			},
			expectedStatus: &dataplane.ConfigurationStatus{
				CorrelationId: toPtr(correlationID),
				Events: &[]dataplane.Events{
					{
						Status:    toPtr(dataplane.SUCCESS),
						Message:   toPtr("Success"),
						Timestamp: nil,
					},
					{
						Status:    toPtr(dataplane.INPROGRESS),
						Message:   toPtr("In Progress"),
						Timestamp: nil,
					},
				},
				InstanceId: &instanceID,
			},
			expectedResponse: 200,
		},
		{
			name:       "not found",
			instanceID: "unknown-instance-id",
			events:     nil,
			expectedStatus: &dataplane.ConfigurationStatus{
				CorrelationId: toPtr(correlationID),
				Events: &[]dataplane.Events{
					{
						Status:    toPtr(dataplane.SUCCESS),
						Message:   toPtr("Success"),
						Timestamp: nil,
					},
					{
						Status:    toPtr(dataplane.INPROGRESS),
						Message:   toPtr("In Progress"),
						Timestamp: nil,
					},
				},
				InstanceId: toPtr("unknown-instance-id"),
			},
			expectedResponse: 404,
		},
	}

	instance := &instances.Instance{InstanceId: instanceID, Type: instances.Type_NGINX, Version: "1.23.1"}
	dataPlaneServer := NewDataPlaneServer(&config.Config{}, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}
	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	tcpAddr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)

	assert.True(t, ok)
	assert.NotNil(t, tcpAddr.Port)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.events != nil {
				dataPlaneServer.configEvents = map[string][]*instances.ConfigurationStatus{
					test.instanceID: test.events,
				}
			}
			res, err := performGetInstanceConfigurationStatusRequest(ctx, dataPlaneServer, test.instanceID)
			require.NoError(t, err)
			assert.Equal(t, test.expectedResponse, res.StatusCode)

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			result := &dataplane.ConfigurationStatus{}
			err = json.Unmarshal(resBody, &result)
			require.NoError(t, err)

			compareResponse(t, result, *test.expectedStatus.Events, test.expectedStatus, resBody)

			res.Body.Close()
		})
	}
}

func compareResponse(t *testing.T, result *dataplane.ConfigurationStatus, events []dataplane.Events,
	expectedStatus *dataplane.ConfigurationStatus, resBody []byte,
) {
	t.Helper()
	if result.Events != nil {
		for key, resultStatus := range *result.Events {
			assert.Equal(t, (events)[key].Status, resultStatus.Status)
			assert.Equal(t, (events)[key].Message, resultStatus.Message)
		}
		assert.Equal(t, expectedStatus.CorrelationId, result.CorrelationId)
	} else {
		result2 := dataplane.ErrorResponse{}
		err := json.Unmarshal(resBody, &result2)
		require.NoError(t, err)
		assert.Equal(t, "Unable to find configuration status", result2.Message)
	}
}

func TestDataPlaneServer_MapStatusEnums(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected dataplane.StatusState
	}{
		{
			name:     "Success type",
			input:    "SUCCESS",
			expected: dataplane.SUCCESS,
		}, {
			name:     "In progress type",
			input:    "IN_PROGRESS",
			expected: dataplane.INPROGRESS,
		}, {
			name:     "Failed type",
			input:    "FAILED",
			expected: dataplane.FAILED,
		}, {
			name:     "Rollback Failed type",
			input:    "ROLLBACK_FAILED",
			expected: dataplane.ROLLBACKFAILED,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := mapStatusEnums(test.input)
			assert.Equal(t, toPtr(test.expected), result)
		})
	}
}

func performPutRequest(
	ctx context.Context,
	dataPlaneServer *DataPlaneServer,
	instanceID string,
	data []byte,
) (*http.Response, error) {
	addr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)
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

func performGetInstanceConfigurationStatusRequest(
	ctx context.Context,
	dataPlaneServer *DataPlaneServer,
	instanceID string,
) (*http.Response, error) {
	addr, ok := dataPlaneServer.server.Addr().(*net.TCPAddr)
	if !ok {
		return nil, fmt.Errorf("unable to get server address")
	}

	target := "http://" + addr.AddrPort().String() +
		"/api/v1/instances/" + instanceID + "/configurations/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	return client.Do(req)
}
