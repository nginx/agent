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
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

var (
	apiHost string
	apiPort string
)

func setupTest(tb testing.TB) func(tb testing.TB) {
	tb.Helper()
	var container testcontainers.Container
	ctx := context.TODO()

	if os.Getenv("TEST_ENV") == "Container" {
		tb.Log("Running tests in a container environment")
		container = utils.StartContainer(
			ctx,
			tb,
			"Processes updated",
		)

		ipAddress, err := container.Host(ctx)
		require.NoError(tb, err)
		ports, err := container.Ports(ctx)
		require.NoError(tb, err)

		apiHost = ipAddress
		apiPort = ports["9091/tcp"][0].HostPort
	} else {
		apiHost = "0.0.0.0"
		apiPort = "9091"
		tb.Log("Running tests on local machine")
	}

	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			logReader, err := container.Logs(ctx)
			require.NoError(tb, err)

			buf, err := io.ReadAll(logReader)
			require.NoError(tb, err)
			logs := string(buf)

			tb.Log(logs)
			assert.NotContains(tb, logs, "level=ERROR", "agent log file contains logs at error level")

			err = container.Terminate(ctx)
			require.NoError(tb, err)
		}
	}
}

func TestDataplaneAPI_GetInstances(t *testing.T) {
	teardownTest := setupTest(t)
	defer teardownTest(t)

	client := resty.New()
	client.SetRetryCount(3).SetRetryWaitTime(50 * time.Millisecond).SetRetryMaxWaitTime(200 * time.Millisecond)

	url := fmt.Sprintf("http://%s/api/v1/instances/", net.JoinHostPort(apiHost, apiPort))
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	var response []*dataplane.Instance

	responseData := resp.Body()
	err = json.Unmarshal(responseData, &response)

	require.NoError(t, err)
	assert.True(t, json.Valid(responseData))

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
