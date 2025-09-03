// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/integration/utils"
)

func (s *MPITestSuite) Test_APIActionRequest_GetEmptyUpstreams() {
	if os.Getenv("IMAGE_PATH") != "/nginx-plus/agent" {
		s.T().Skip("Skipping TestGrpc_Test4_APIActionRequest since image is not NGINX Plus")
	}

	newConfigFile := "../../config/nginx/nginx-plus-with-test-location.conf"

	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		newConfigFile,
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].GetCommandResponse().GetMessage() < responses[j].GetCommandResponse().GetMessage()
	})

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())

	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)

	request := fmt.Sprintf(`{
    "message_meta": {
        "message_id": "5d0fa83e-351c-4009-90cd-1f2acce2d184",
        "correlation_id": "79794c1c-8e91-47c1-a92c-b9a0c3f1a263",
        "timestamp": "2023-01-15T01:30:15.01Z"
    },
    "action_request": {
        "instance_id": "%s",
        "nginx_plus_action": {
            "get_upstreams": {}
        }
    }
}`, s.nginxInstanceID)

	client := resty.New()
	client.SetRetryCount(utils.RetryCount).SetRetryWaitTime(utils.RetryWaitTime).SetRetryMaxWaitTime(
		utils.RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", utils.MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	responses = utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("{}", responses[0].GetCommandResponse().GetMessage())
}
