// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/integration/managementplane/configs"
	"github.com/testcontainers/testcontainers-go"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	RetryCount       = 10
	RetryWaitTime    = 5 * time.Second
	RetryMaxWaitTime = 1 * time.Minute
	permissions      = 0o666
)

var (
	MockManagementPlaneAPIAddress          string
	AuxiliaryMockManagementPlaneAPIAddress string
)

func PerformConfigApply(t *testing.T, nginxInstanceID, mockManagementPlaneAPIAddress string) {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/instance/%s/config/apply", mockManagementPlaneAPIAddress, nginxInstanceID)
	resp, err := client.R().EnableTrace().Post(url)

	t.Logf("Config ApplyResponse: %s", resp.String())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func CurrentFileOverview(t *testing.T, nginxInstanceID, mockManagementPlaneAPIAddress string) *mpi.FileOverview {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/instance/%s/config", mockManagementPlaneAPIAddress, nginxInstanceID)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()

	overview := mpi.GetOverviewResponse{}

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	unmarshalErr := pb.Unmarshal(responseData, &overview)
	require.NoError(t, unmarshalErr)

	return overview.GetOverview()
}

func PerformInvalidConfigApply(t *testing.T, nginxInstanceID string) {
	t.Helper()

	client := resty.New()

	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

	body := fmt.Sprintf(`{
			"message_meta": {
				"message_id": "e2254df9-8edd-4900-91ce-88782473bcb9",
				"correlation_id": "9673f3b4-bf33-4d98-ade1-ded9266f6818",
				"timestamp": "2023-01-15T01:30:15.01Z"
			},
			"config_apply_request": {
				"overview": {
					"files": [{
						"file_meta": {
							"name": "/etc/nginx/nginx.conf",
							"hash": "ea57e443-e968-3a50-b842-f37112acde71",
							"modifiedTime": "2023-01-15T01:30:15.01Z",
							"permissions": "0644",
							"size": 0
						},
						"action": "FILE_ACTION_UPDATE"
					},
					{
						"file_meta": {
							"name": "/unknown/nginx.conf",
							"hash": "bd1f337d-6874-35ea-9d4d-2b543efd42cf",
							"modifiedTime": "2023-01-15T01:30:15.01Z",
							"permissions": "0644",
							"size": 0
						},
						"action": "FILE_ACTION_ADD"
					}],
					"config_version": {
						"instance_id": "%s",
						"version": "6f343257-55e3-309e-a2eb-bb13af5f80f4"
					}
				}
			}
		}`, nginxInstanceID)
	url := fmt.Sprintf("http://%s/api/v1/requests", MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(body).Post(url)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func CheckManifestFile(t *testing.T, container testcontainers.Container,
	expectedContent map[string]*model.ManifestFile,
) {
	t.Helper()
	ctx := context.Background()
	file, err := container.CopyFileFromContainer(ctx, "/var/lib/nginx-agent/manifest.json")
	require.NoError(t, err)
	fileContent, err := io.ReadAll(file)
	require.NoError(t, err)

	var manifestFiles map[string]*model.ManifestFile

	err = json.Unmarshal(fileContent, &manifestFiles)
	assert.NotEmpty(t, fileContent)
	require.NoError(t, err)

	assert.Equal(t, expectedContent, manifestFiles)
}

func WriteConfigFileMock(t *testing.T, nginxInstanceID, file1, file2, file3 string) {
	t.Helper()
	tempDir := t.TempDir()

	file := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	t.Logf("File: %s", file.Name())

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		writeErr := os.WriteFile(file.Name(), []byte(configs.NginxPlusConfigWithMultipleInclude(
			file1, file2, file3)), permissions)
		require.NoError(t, writeErr)
	} else {
		writeErr := os.WriteFile(file.Name(), []byte(configs.NginxConfigWithMultipleInclude(
			file1, file2, file3)), permissions)
		require.NoError(t, writeErr)
	}

	err := MockManagementPlaneGrpcContainer.CopyFileToContainer(
		t.Context(),
		file.Name(),
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", nginxInstanceID),
		permissions,
	)
	require.NoError(t, err)
}

func WriteConfigFileDataplane(t *testing.T, file1, file2, file3 string) {
	t.Helper()
	tempDir := t.TempDir()

	file := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	t.Logf("File: %s", file.Name())

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		writeErr := os.WriteFile(file.Name(), []byte(configs.NginxPlusConfigWithMultipleInclude(
			file1, file2, file3)), permissions)
		require.NoError(t, writeErr)
	} else {
		writeErr := os.WriteFile(file.Name(), []byte(configs.NginxConfigWithMultipleInclude(
			file1, file2, file3)), permissions)
		require.NoError(t, writeErr)
	}

	err := Container.CopyFileToContainer(
		t.Context(),
		file.Name(),
		"/etc/nginx/nginx.conf",
		permissions,
	)
	require.NoError(t, err)
}
