// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

func (s *ConfigApplyTestSuite) TestGrpc_Test1_Reconnection() {
	timeout := 15 * time.Second

	stopErr := utils.MockManagementPlaneGrpcContainer.Stop(s.ctx, &timeout)

	s.Require().NoError(stopErr)

	startErr := utils.MockManagementPlaneGrpcContainer.Start(s.ctx)
	s.Require().NoError(startErr)

	ipAddress, err := utils.MockManagementPlaneGrpcContainer.Host(s.ctx)
	s.Require().NoError(err)
	ports, err := utils.MockManagementPlaneGrpcContainer.Ports(s.ctx)
	s.Require().NoError(err)
	utils.MockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9093/tcp"][0].HostPort)

	currentID := utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.Equal(s.nginxInstanceID, currentID)
}

// Verify that the agent sends a connection request and an update data plane status request
func (s *MPITestSuite) TestGrpc_Test2_StartUp() {
	utils.VerifyUpdateDataPlaneHealth(s.T(), utils.MockManagementPlaneAPIAddress)
}

func (s *MPITestSuite) TestGrpc_Test3_DataplaneHealthRequest() {
	request := `{
			"message_meta": {
				"message_id": "5d0fa83e-351c-4009-90cd-1f2acce2d184",
				"correlation_id": "79794c1c-8e91-47c1-a92c-b9a0c3f1a263",
				"timestamp": "2023-01-15T01:30:15.01Z"
			},
			"health_request": {}
		}`

	client := resty.New()
	client.SetRetryCount(utils.RetryCount).SetRetryWaitTime(utils.RetryWaitTime).SetRetryMaxWaitTime(
		utils.RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", utils.MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully sent health status update", responses[0].GetCommandResponse().GetMessage())
}
