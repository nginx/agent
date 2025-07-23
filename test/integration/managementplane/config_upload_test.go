// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/suite"
)

type MPITestSuite struct {
	suite.Suite
	ctx             context.Context
	teardownTest    func(testing.TB)
	nginxInstanceID string
}

func (s *MPITestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

func (s *MPITestSuite) TearDownTest() {
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
}

func (s *MPITestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), true, false, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	s.False(s.T().Failed())
}

func (s *MPITestSuite) TestConfigUpload() {
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
}`, s.nginxInstanceID)

	s.T().Logf("Sending config upload request: %s", request)

	client := resty.New()
	client.SetRetryCount(utils.RetryCount).SetRetryWaitTime(utils.RetryWaitTime).SetRetryMaxWaitTime(
		utils.RetryMaxWaitTime)

	url := fmt.Sprintf("http://%s/api/v1/requests", utils.MockManagementPlaneAPIAddress)
	resp, err := client.R().EnableTrace().SetBody(request).Post(url)

	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode())

	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
}

func TestMPITestSuite(t *testing.T) {
	suite.Run(t, new(MPITestSuite))
}
