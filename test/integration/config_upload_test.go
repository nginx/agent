// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpc_ConfigUpload(t *testing.T) {
	teardownTest := SetupConnectionTest(t, true, false)
	defer teardownTest(t)

	nginxInstanceID := VerifyConnection(t, 2)
	assert.False(t, t.Failed())

	responses := GetManagementPlaneResponses(t, 1)

	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	request := fmt.Sprintf(`{
	"message_meta": {
		"message_id": "5d0fa83e-351c-4009-90cd-1f2acce2d184",
		"correlation_id": "79794c1c-8e91-47c1-a92c-b9a0c3f1a263",
		"timestamp": "2023-01-15T01:30:15.01Z"
	},
	"config_upload_request": {
      "overview" : {
        "config_version": {
          "instance_id": "%s"
        }
      }
	}
}`, nginxInstanceID)

	t.Logf("Sending config upload request: %s", request)

	client := resty.New()
	client.SetRetryCount(RetryCount).SetRetryWaitTime(RetryWaitTime).SetRetryMaxWaitTime(RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responses = GetManagementPlaneResponses(t, 2)

	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	assert.Equal(t, "Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
}
