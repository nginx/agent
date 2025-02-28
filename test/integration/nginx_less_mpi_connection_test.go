// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stretchr/testify/assert"
)

// Verify that the agent sends a connection request to Management Plane even when Nginx is not present
func TestNginxLessGrpc_Connection(t *testing.T) {
	teardownTest := setupConnectionTest(t, true, true)
	defer teardownTest(t)

	verifyNginxLessConnection(t, 1)
	assert.False(t, t.Failed())
}

// nolint
func verifyNginxLessConnection(t *testing.T, instancesLength int) string {
	t.Helper()

	client := resty.New()
	client.SetRetryCount(retryCount).SetRetryWaitTime(retryWaitTime).SetRetryMaxWaitTime(retryMaxWaitTime)
	connectionRequest := mpi.CreateConnectionRequest{}

	url := fmt.Sprintf("http://%s/api/v1/connection", mockManagementPlaneAPIAddress)
	t.Logf("Connecting to %s", url)
	resp, err := client.R().EnableTrace().Get(url)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())

	responseData := resp.Body()
	t.Logf("Response: %s", string(responseData))
	assert.True(t, json.Valid(responseData))

	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	unmarshalErr := pb.Unmarshal(responseData, &connectionRequest)
	require.NoError(t, unmarshalErr)

	t.Logf("ConnectionRequest: %v", &connectionRequest)

	resource := connectionRequest.GetResource()

	assert.NotNil(t, resource.GetResourceId())
	assert.NotNil(t, resource.GetContainerInfo().GetContainerId())

	assert.Len(t, resource.GetInstances(), instancesLength)

	var nginxInstanceID string

	for _, instance := range resource.GetInstances() {
		switch instance.GetInstanceMeta().GetInstanceType() {
		case mpi.InstanceMeta_INSTANCE_TYPE_AGENT:
			agentInstanceMeta := instance.GetInstanceMeta()

			assert.NotEmpty(t, agentInstanceMeta.GetInstanceId())
			assert.NotEmpty(t, agentInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx-agent/nginx-agent.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX:
			nginxInstanceMeta := instance.GetInstanceMeta()

			nginxInstanceID = nginxInstanceMeta.GetInstanceId()
			assert.NotEmpty(t, nginxInstanceID)
			assert.NotEmpty(t, nginxInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx/nginx.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS:
			nginxInstanceMeta := instance.GetInstanceMeta()

			nginxInstanceID = nginxInstanceMeta.GetInstanceId()
			assert.NotEmpty(t, nginxInstanceID)
			assert.NotEmpty(t, nginxInstanceMeta.GetVersion())

			assert.NotEmpty(t, instance.GetInstanceRuntime().GetBinaryPath())

			assert.Equal(t, "/etc/nginx/nginx.conf", instance.GetInstanceRuntime().GetConfigPath())
		case mpi.InstanceMeta_INSTANCE_TYPE_UNIT,
			mpi.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED:
			fallthrough
		default:
			t.Fail()
		}
	}

	return nginxInstanceID
}
