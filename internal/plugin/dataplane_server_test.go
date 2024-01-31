/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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

	"github.com/nginx/agent/v3/api/grpc/instances"
	http_api "github.com/nginx/agent/v3/api/http"
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
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, dataplaneServer.server.Addr().(*net.TCPAddr).Port)
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
	assert.NoError(t, err)
	go messagePipe.Run()

	messagePipe.Process(&bus.Message{Topic: bus.INSTANCES_TOPIC, Data: testInstances})

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, testInstances, dataplaneServer.instances)
}

func TestDataplaneServer_GetInstances(t *testing.T) {
	expected := &http_api.Instance{InstanceId: toPtr("ae6c58c1-bc92-30c1-a9c9-85591422068e"), Type: toPtr(http_api.NGINX), Version: toPtr("1.23.1")}
	instance := &instances.Instance{InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e", Type: instances.Type_NGINX, Version: "1.23.1"}

	instanceService := &servicefakes.FakeInstanceServiceInterface{}

	dataplaneServer := NewDataplaneServer(&DataplaneServerParameters{
		Host:            "",
		Port:            0,
		Logger:          slog.Default(),
		instanceService: instanceService,
	})

	dataplaneServer.instances = []*instances.Instance{instance}

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

	result := []*http_api.Instance{}
	err = json.Unmarshal(resBody, &result)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, expected, result[0])
}

func TestDataplaneServer_UpdateInstanceConfiguration(t *testing.T) {
	unknownInstanceId := "fe4c58c1-bc92-30c1-a9c9-85591422068e"
	instanceId := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	data := []byte(`{"location": "http://file-server.com"}`)
	instance := &instances.Instance{InstanceId: instanceId, Type: instances.Type_NGINX, Version: "1.23.1"}

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
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NotNil(t, dataplaneServer.server.Addr().(*net.TCPAddr).Port)

	tests := []struct {
		name               string
		instanceId         string
		expectedStatusCode int
		expectedMessage    string
	}{
		{
			name:               "Update known instance configuration",
			instanceId:         instanceId,
			expectedStatusCode: 200,
		},
		{
			name:               "Update unknown instance configuration",
			instanceId:         unknownInstanceId,
			expectedStatusCode: 404,
			expectedMessage:    fmt.Sprintf("Unable to find instance %s", unknownInstanceId),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			res, err := performPutRequest(dataplaneServer, test.instanceId, data)
			assert.NoError(tt, err)
			assert.Equal(tt, test.expectedStatusCode, res.StatusCode)

			resBody, err := io.ReadAll(res.Body)
			assert.NoError(tt, err)

			if test.expectedMessage == "" {
				result := http_api.CorrelationId{}
				err = json.Unmarshal(resBody, &result)
				assert.NoError(tt, err)
				assert.NotEmpty(tt, result.CorrelationId)
			} else {
				result := http_api.ErrorResponse{}
				err = json.Unmarshal(resBody, &result)
				assert.NoError(tt, err)
				assert.Equal(tt, test.expectedMessage, result.Message)
			}
		})
	}
}

func TestDataplaneServer_GetInstanceConfigurationStatus(t *testing.T) {
	instanceId := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	correlationId := "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	status := &instances.ConfigurationStatus{
		InstanceId:    instanceId,
		CorrelationId: correlationId,
		Status:        instances.Status_SUCCESS,
		Message:       "Success",
	}
	instance := &instances.Instance{InstanceId: instanceId, Type: instances.Type_NGINX, Version: "1.23.1"}

	instanceService := &servicefakes.FakeInstanceServiceInterface{}
	instanceService.GetInstancesReturns([]*instances.Instance{instance})

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

	dataplaneServer.configurationStatues = map[string]*instances.ConfigurationStatus{
		instanceId: status,
	}

	// Test happy path
	res, err := performGetInstanceConfigurationStatusRequest(dataplaneServer, instanceId)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	assert.NoError(t, err)

	result := &http_api.ConfigurationStatus{}
	err = json.Unmarshal(resBody, &result)
	assert.NoError(t, err)
	assert.Equal(t, toPtr(status.CorrelationId), result.CorrelationId)
	assert.Equal(t, toPtr(status.Message), result.Message)
	assert.Equal(t, toPtr(http_api.SUCCESS), result.Status)
	assert.NotNil(t, result.LastUpdated)

	// Test configuration status not found
	res, err = performGetInstanceConfigurationStatusRequest(dataplaneServer, "unknown-instance-id")
	assert.NoError(t, err)
	assert.Equal(t, 404, res.StatusCode)

	resBody, err = io.ReadAll(res.Body)
	assert.NoError(t, err)

	result2 := http_api.ErrorResponse{}
	err = json.Unmarshal(resBody, &result2)
	assert.NoError(t, err)
	assert.Equal(t, "Unable to find configuration status", result2.Message)
}

func TestDataplaneServer_MapStatusEnums(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected http_api.ConfigurationStatusType
	}{
		{
			name:     "Success type",
			input:    "SUCCESS",
			expected: http_api.SUCCESS,
		}, {
			name:     "In progress type",
			input:    "IN_PROGRESS",
			expected: http_api.INPROGESS,
		}, {
			name:     "Failed type",
			input:    "FAILED",
			expected: http_api.FAILED,
		}, {
			name:     "Rollback Failed type",
			input:    "ROLLBACK_FAILED",
			expected: http_api.ROLLBACKFAILED,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			result := mapStatusEnums(test.input)
			assert.Equal(t, toPtr(test.expected), result)
		})
	}
}

func performPutRequest(dataplaneServer *DataplaneServer, instanceId string, data []byte) (*http.Response, error) {
	target := "http://" + dataplaneServer.server.Addr().(*net.TCPAddr).AddrPort().String() + "/api/v1/instances/" + instanceId + "/configurations"
	req, err := http.NewRequest(http.MethodPut, target, bytes.NewBuffer(data))
	if err != nil {
		return &http.Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

func performGetInstanceConfigurationStatusRequest(dataplaneServer *DataplaneServer, instanceId string) (*http.Response, error) {
	target := "http://" + dataplaneServer.server.Addr().(*net.TCPAddr).AddrPort().String() + "/api/v1/instances/" + instanceId + "/configurations/status"
	return http.Get(target)
}
