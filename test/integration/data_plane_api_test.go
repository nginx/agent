// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/test/config"
	"github.com/nginx/agent/v3/test/helpers"

	mockHttp "github.com/nginx/agent/v3/test/mock/http"
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
	ctx := context.Background()

	configDir := tb.TempDir()
	dir := filepath.Join(configDir, "/etc/nginx/")
	helpers.CreateDirWithErrorCheck(tb, dir)

	content := config.GetNginxConfWithTestLocation()

	nginxConfigFilePath := filepath.Join(dir, "nginx.conf")
	err := os.WriteFile(nginxConfigFilePath, []byte(content), 0o600)
	require.NoError(tb, err)
	defer helpers.RemoveFileWithErrorCheck(tb, nginxConfigFilePath)

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

		mockManagementPlaneContainer = helpers.StartMockManagementPlaneHTTPContainer(
			ctx,
			tb,
			containerNetwork,
			nginxConfigFilePath,
		)

		mockManagementPlaneAddress = "managementPlane:9092"
		tb.Logf("Mock management server running on %s", mockManagementPlaneAddress)

		params := &helpers.Parameters{
			NginxConfigPath:      "../config/nginx/nginx.conf",
			NginxAgentConfigPath: "../config/agent/nginx-agent-with-data-plane-api.conf",
			LogMessage:           "Processes updated",
		}

		container = helpers.StartContainer(
			ctx,
			tb,
			containerNetwork,
			params,
		)

		ipAddress, err := container.Host(ctx)
		require.NoError(tb, err)
		ports, err := container.Ports(ctx)
		require.NoError(tb, err)

		apiHost = ipAddress
		apiPort = ports["9091/tcp"][0].HostPort
	} else {
		server := mockHttp.NewManagementServer(configDir)
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
			helpers.LogAndTerminateContainers(ctx, tb, mockManagementPlaneContainer, container)
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
