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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestDataPlaneServer_Init(t *testing.T) {
	agentConfig := types.GetAgentConfig()
	dataPlaneServer := NewDataPlaneServer(agentConfig, slog.Default())

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr := dataPlaneServer.server.Addr
	assert.NotNil(t, addr)

	err = dataPlaneServer.Close()
	require.NoError(t, err)
}

func TestDataPlaneServer_Process(t *testing.T) {
	agentConfig := types.GetAgentConfig()
	dataPlaneServer := NewDataPlaneServer(agentConfig, slog.Default())

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
			name:  "Test 1: Instances test",
			data:  []*instances.Instance{{InstanceId: instanceID, Type: instances.Type_NGINX}},
			topic: bus.InstancesTopic,
		},
		{
			name: "Test 2: Instances complete update",
			data: &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_SUCCESS,
				Message:       "config updated",
			},
			topic: bus.InstanceConfigUpdateTopic,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			messagePipe.Process(&bus.Message{Topic: tt.topic, Data: tt.data})
			time.Sleep(10 * time.Millisecond)

			var actual interface{}
			if tt.topic == bus.InstancesTopic {
				actual = dataPlaneServer.instances
			} else {
				actual = dataPlaneServer.configEvents[instanceID][0]
			}

			assert.Equal(t, tt.data, actual)
		})
	}

	err = dataPlaneServer.Close()
	require.NoError(t, err)
}

func TestDataPlaneServer_GetInstances(t *testing.T) {
	ctx := context.TODO()
	expected := &dataplane.Instance{
		InstanceId: toPtr(instanceID),
		Type:       toPtr(dataplane.NGINX),
		Version:    toPtr("1.23.1"),
	}

	instance := &instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
		Version:    "1.23.1",
	}

	agentConfig := types.GetAgentConfig()
	dataPlaneServer := NewDataPlaneServer(agentConfig, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(ctx, 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr := dataPlaneServer.server.Addr
	assert.NotNil(t, addr)

	target := fmt.Sprintf("http://%s/api/v1/instances", addr)

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

	err = dataPlaneServer.Close()
	require.NoError(t, err)
}

func TestDataPlaneServer_UpdateInstanceConfiguration(t *testing.T) {
	ctx := context.TODO()
	unknownInstanceID := "fe4c58c1-bc92-30c1-a9c9-85591422068e"
	data := []byte(`{"location": "http://file-server.com"}`)
	instance := &instances.Instance{InstanceId: instanceID, Type: instances.Type_NGINX, Version: "1.23.1"}

	agentConfig := types.GetAgentConfig()
	dataPlaneServer := NewDataPlaneServer(agentConfig, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	addr := dataPlaneServer.server.Addr
	assert.NotNil(t, addr)

	tests := []struct {
		name               string
		instanceID         string
		expectedStatusCode int
		expectedMessage    string
	}{
		{
			name:               "Test 1: Update known instance configuration",
			instanceID:         instanceID,
			expectedStatusCode: 200,
		},
		{
			name:               "Test 2: Update unknown instance configuration",
			instanceID:         unknownInstanceID,
			expectedStatusCode: 404,
			expectedMessage:    fmt.Sprintf("Unable to find instance %s", unknownInstanceID),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			res, loopErr := performPutRequest(ctx, dataPlaneServer, test.instanceID, data)
			require.NoError(tt, loopErr)
			assert.Equal(tt, test.expectedStatusCode, res.StatusCode)

			resBody, loopErr := io.ReadAll(res.Body)
			require.NoError(tt, loopErr)

			loopErr = res.Body.Close()
			require.NoError(t, loopErr)

			if test.expectedMessage == "" {
				result := dataplane.CorrelationId{}
				loopErr = json.Unmarshal(resBody, &result)
				require.NoError(tt, loopErr)
				assert.NotEmpty(tt, result.CorrelationId)
			} else {
				result := dataplane.ErrorResponse{}
				loopErr = json.Unmarshal(resBody, &result)
				require.NoError(tt, loopErr)
				assert.Equal(tt, test.expectedMessage, result.Message)
			}
		})
	}

	err = dataPlaneServer.Close()
	require.NoError(t, err)
}

func TestDataPlaneServer_GetInstanceConfigurationStatus(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name             string
		instanceID       string
		events           []*instances.ConfigurationStatus
		expectedStatus   *dataplane.ConfigurationStatus
		expectedResponse int
	}{
		{
			name:       "Test 1: Successful config update",
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
				Events: &[]dataplane.Event{
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
				InstanceId: toPtr(instanceID),
			},
			expectedResponse: 200,
		},
		{
			name:       "Test 2: Instance ID not found",
			instanceID: "unknown-instance-id",
			events:     nil,
			expectedStatus: &dataplane.ConfigurationStatus{
				CorrelationId: toPtr(correlationID),
				Events: &[]dataplane.Event{
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
	agentConfig := types.GetAgentConfig()
	dataPlaneServer := NewDataPlaneServer(agentConfig, slog.Default())
	dataPlaneServer.instances = []*instances.Instance{instance}
	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	err := messagePipe.Register(100, []bus.Plugin{dataPlaneServer})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	tcpAddr := dataPlaneServer.server.Addr

	assert.NotNil(t, tcpAddr)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.events != nil {
				dataPlaneServer.configEvents = map[string][]*instances.ConfigurationStatus{
					test.instanceID: test.events,
				}
			}
			res, loopErr := performGetInstanceConfigurationStatusRequest(ctx, dataPlaneServer, test.instanceID)
			require.NoError(t, loopErr)
			assert.Equal(t, test.expectedResponse, res.StatusCode)

			resBody, loopErr := io.ReadAll(res.Body)
			require.NoError(t, loopErr)

			result := &dataplane.ConfigurationStatus{}
			loopErr = json.Unmarshal(resBody, &result)
			require.NoError(t, loopErr)

			compareResponse(t, result, *test.expectedStatus.Events, test.expectedStatus, resBody)

			res.Body.Close()
		})
	}

	err = dataPlaneServer.Close()
	require.NoError(t, err)
}

func compareResponse(t *testing.T, result *dataplane.ConfigurationStatus, events []dataplane.Event,
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
			name:     "Test 1: Success type",
			input:    "SUCCESS",
			expected: dataplane.SUCCESS,
		}, {
			name:     "Test 2: In progress type",
			input:    "IN_PROGRESS",
			expected: dataplane.INPROGRESS,
		}, {
			name:     "Test 3: Failed type",
			input:    "FAILED",
			expected: dataplane.FAILED,
		}, {
			name:     "Test 4: Rollback Failed type",
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
	addr := dataPlaneServer.server.Addr
	if addr == "" {
		return nil, fmt.Errorf("unable to get server address")
	}
	target := fmt.Sprintf("http://%s/api/v1/instances/%s/configurations", addr, instanceID)
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
	addr := dataPlaneServer.server.Addr
	if addr == "" {
		return nil, fmt.Errorf("unable to get server address")
	}

	target := fmt.Sprintf("http://%s/api/v1/instances/%s/configurations/status", addr, instanceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return &http.Response{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	return client.Do(req)
}
