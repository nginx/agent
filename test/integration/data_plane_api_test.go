// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/test"
	"github.com/nginx/agent/v3/test/config"

	"github.com/nginx/agent/v3/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

var (
	apiHost                      string
	apiPort                      string
	instancesURL                 string
	mockManagementPlaneContainer testcontainers.Container
	mockManagementPlaneAddress   string
)

func setupTest(tb testing.TB) func(tb testing.TB) {
	tb.Helper()
	var container testcontainers.Container
	ctx := context.TODO()

	configDir := tb.TempDir()
	dir := filepath.Join(configDir, "/etc/nginx/")
	test.CreateDirWithErrorCheck(tb, dir)

	content, err := config.GetNginxConfWithTestLocation()
	require.NoError(tb, err)

	nginxConfigFilePath := filepath.Join(dir, "nginx.conf")
	err = os.WriteFile(nginxConfigFilePath, []byte(content), 0o600)
	require.NoError(tb, err)
	defer test.RemoveFileWithErrorCheck(tb, nginxConfigFilePath)

	if os.Getenv("TEST_ENV") == "Container" {
		tb.Log("Running tests in a container environment")

		containerNetwork, err := network.New(
			ctx,
			network.WithCheckDuplicate(),
			network.WithAttachable(),
		)
		require.NoError(tb, err)
		tb.Cleanup(func() {
			require.NoError(tb, containerNetwork.Remove(ctx))
		})

		mockManagementPlaneContainer = test.StartMockManagementPlaneContainer(
			ctx,
			tb,
			containerNetwork,
			nginxConfigFilePath,
		)

		mockManagementPlaneAddress = "managementPlane:9092"
		tb.Logf("Mock management server running on %s", mockManagementPlaneAddress)

		container = test.StartContainer(
			ctx,
			tb,
			containerNetwork,
			"Processes updated",
			"../config/nginx/nginx.conf",
		)

		ipAddress, err := container.Host(ctx)
		require.NoError(tb, err)
		ports, err := container.Ports(ctx)
		require.NoError(tb, err)

		apiHost = ipAddress
		apiPort = ports["9091/tcp"][0].HostPort
	} else {
		server := mock.NewManagementServer(configDir)
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(tb, err)

		go server.StartServer(listener)

		mockManagementPlaneAddress = listener.Addr().String()

		apiHost = "localhost"
		apiPort = "8038"
		tb.Log("Running tests on local machine")
	}

	instancesURL = fmt.Sprintf("http://%s/api/v1/instances/", net.JoinHostPort(apiHost, apiPort))

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			tb.Log("Logging mock management container logs")
			logReader, err := mockManagementPlaneContainer.Logs(ctx)
			require.NoError(tb, err)

			buf, err := io.ReadAll(logReader)
			require.NoError(tb, err)
			logs := string(buf)

			tb.Log(logs)

			err = mockManagementPlaneContainer.Terminate(ctx)
			require.NoError(tb, err)

			tb.Log("Logging nginx agent container logs")
			logReader, err = container.Logs(ctx)
			require.NoError(tb, err)

			buf, err = io.ReadAll(logReader)
			require.NoError(tb, err)
			logs = string(buf)

			tb.Log(logs)
			assert.NotContains(tb, logs, "level=ERROR", "agent log file contains logs at error level")

			err = container.Terminate(ctx)
			require.NoError(tb, err)
		}
	}
}

func TestDataPlaneAPI_GetInstances(t *testing.T) {
	teardownTest := setupTest(t)
	defer teardownTest(t)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	resp, err := client.R().EnableTrace().Get(instancesURL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var response []*dataplane.Instance

	responseData := resp.Body()
	assert.True(t, json.Valid(responseData))

	err = json.Unmarshal(responseData, &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)
	assert.Equal(t, dataplane.NGINX, *response[0].Type)
	assert.NotNil(t, response[0].Version)
	assert.NotNil(t, response[0].InstanceId)
	assert.NotNil(t, response[0].Meta)

	nginxMeta, err := response[0].Meta.AsNginxMeta()
	require.NoError(t, err)
	assert.NotNil(t, nginxMeta.ConfPath)
	assert.NotNil(t, nginxMeta.ExePath)
	assert.Equal(t, dataplane.MetaType("NginxMeta"), nginxMeta.Type)
}

func TestDataplaneAPI_UpdateInstances(t *testing.T) {
	teardownTest := setupTest(t)
	defer teardownTest(t)

	body := &dataplane.UpdateInstanceConfigurationJSONRequestBody{
		Location: test.ToPtr(fmt.Sprintf("http://%s/api/v1", mockManagementPlaneAddress)),
	}

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	instanceID := getInstanceID(t, client)

	url := fmt.Sprintf("%s%s/configurations", instancesURL, instanceID)
	resp, err := client.R().EnableTrace().SetBody(body).Put(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var response *dataplane.CorrelationId

	responseData := resp.Body()
	assert.True(t, json.Valid(responseData))

	err = json.Unmarshal(responseData, &response)
	require.NoError(t, err)

	assert.NotNil(t, response.CorrelationId)

ConfigStatus:
	for {
		statusResponse := getConfigurationStatus(t, client, instanceID)
		for _, event := range *statusResponse.Events {
			t.Log(*event.Status)
			t.Log(*event.Message)
			if *event.Status != dataplane.INPROGRESS {
				assert.Equal(t, "Config applied successfully", *event.Message)
				assert.Equal(t, test.ToPtr(dataplane.SUCCESS), event.Status)

				break ConfigStatus
			}
			assert.Equal(t, "Instance configuration update in progress", *event.Message)
			assert.Equal(t, test.ToPtr(dataplane.INPROGRESS), event.Status)
		}

		time.Sleep(time.Second)
	}
}

func getConfigurationStatus(t *testing.T, client *resty.Client, instanceID string) *dataplane.ConfigurationStatus {
	t.Helper()
	url := fmt.Sprintf("%s%s/configurations/status", instancesURL, instanceID)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return err != nil || r.StatusCode() == http.StatusNotFound
	})
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var statusResponse *dataplane.ConfigurationStatus

	responseData := resp.Body()
	assert.True(t, json.Valid(responseData))

	err = json.Unmarshal(responseData, &statusResponse)
	require.NoError(t, err)

	assert.NotNil(t, statusResponse.CorrelationId)
	for _, event := range *statusResponse.Events {
		assert.NotNil(t, event.Timestamp)
		assert.NotNil(t, event.Message)
		assert.NotNil(t, event.Status)
	}

	return statusResponse
}

func getInstanceID(t *testing.T, client *resty.Client) string {
	t.Helper()

	resp, err := client.R().EnableTrace().Get(instancesURL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var instancesResponse []*dataplane.Instance

	responseData := resp.Body()
	assert.True(t, json.Valid(responseData))

	err = json.Unmarshal(responseData, &instancesResponse)
	require.NoError(t, err)

	assert.Len(t, instancesResponse, 1)
	assert.NotNil(t, instancesResponse[0].InstanceId)
	instanceID := instancesResponse[0].InstanceId

	return *instanceID
}
