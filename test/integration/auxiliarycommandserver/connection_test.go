// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package auxiliarycommandserver

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"net"
	"net/http"
	"testing"
	"time"
)

type AuxiliaryTestSuite struct {
	suite.Suite
	teardownTest func(tb testing.TB)
	instanceID   string
}

func (s *AuxiliaryTestSuite) SetupSuite() {
	t := s.T()
	// Expect errors in logs should be false for recconnection tests
	// For now for these test we will skip checking the logs for errors
	s.teardownTest = utils.SetupConnectionTest(t, false, false, true,
		"../../config/agent/nginx-agent-with-auxiliary-command.conf")
}

func (s *AuxiliaryTestSuite) TearDownSuite() {

	s.teardownTest(s.T())
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(AuxiliaryTestSuite))
}

func (s *AuxiliaryTestSuite) TestAuxiliary_Connection() {

	s.instanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	assert.False(s.T(), s.T().Failed())
	utils.VerifyUpdateDataPlaneHealth(s.T(), utils.MockManagementPlaneAPIAddress)

	utils.VerifyConnection(s.T(), 2, utils.AuxiliaryMockManagementPlaneAPIAddress)
	assert.False(s.T(), s.T().Failed())
	utils.VerifyUpdateDataPlaneHealth(s.T(), utils.AuxiliaryMockManagementPlaneAPIAddress)

}

func (s *AuxiliaryTestSuite) TestAuxiliary_Reconnection() {
	ctx := context.Background()
	timeout := 15 * time.Second

	originalID := utils.VerifyConnection(s.T(), 2, utils.AuxiliaryMockManagementPlaneAPIAddress)
	stopErr := utils.AuxiliaryMockManagementPlaneGrpcContainer.Stop(context.Background(), &timeout)

	require.NoError(s.T(), stopErr)

	utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	assert.False(s.T(), s.T().Failed())

	startErr := utils.AuxiliaryMockManagementPlaneGrpcContainer.Start(ctx)
	require.NoError(s.T(), startErr)

	ipAddress, err := utils.AuxiliaryMockManagementPlaneGrpcContainer.Host(ctx)
	require.NoError(s.T(), err)
	ports, err := utils.AuxiliaryMockManagementPlaneGrpcContainer.Ports(ctx)
	require.NoError(s.T(), err)
	utils.AuxiliaryMockManagementPlaneAPIAddress = net.JoinHostPort(ipAddress, ports["9096/tcp"][0].HostPort)

	currentID := utils.VerifyConnection(s.T(), 2, utils.AuxiliaryMockManagementPlaneAPIAddress)
	assert.Equal(s.T(), originalID, currentID)
}

func (s *AuxiliaryTestSuite) TestAuxiliary_DataplaneHealthRequest() {

	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
	utils.ClearManagementPlaneResponses(s.T(), utils.AuxiliaryMockManagementPlaneAPIAddress)

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

	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode())

	// Check command server has 2 ManagementPlaneResponses as it has sent the request
	commandResponses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, commandResponses[0].GetCommandResponse().GetStatus())
	assert.Equal(s.T(), "Successfully sent health status update", commandResponses[0].GetCommandResponse().GetMessage())
	assert.False(s.T(), s.T().Failed())

	// Check auxiliary server still only has 1 ManagementPlaneResponses as it didn't sent the request
	utils.ManagementPlaneResponses(s.T(), 0, utils.AuxiliaryMockManagementPlaneAPIAddress)
	assert.False(s.T(), s.T().Failed())
}

func (s *AuxiliaryTestSuite) TestAuxiliary_FileWatcher() {
	// Clear any previous responses from previous tests
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
	utils.ClearManagementPlaneResponses(s.T(), utils.AuxiliaryMockManagementPlaneAPIAddress)
	ctx := context.Background()

	err := utils.Container.CopyFileToContainer(
		ctx,
		"../../config/nginx/nginx-with-server-block-access-log.conf",
		"/etc/nginx/nginx.conf",
		0o666,
	)
	require.NoError(s.T(), err)

	// Check command server has 2 ManagementPlaneResponses from updating a file on disk
	commandResponses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, commandResponses[0].GetCommandResponse().GetStatus())
	assert.Equal(s.T(), "Successfully updated all files", commandResponses[0].GetCommandResponse().GetMessage())

	// Check auxiliary server has 2 ManagementPlaneResponses from updating a file on disk
	auxResponses := utils.ManagementPlaneResponses(s.T(), 1, utils.AuxiliaryMockManagementPlaneAPIAddress)
	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, auxResponses[0].GetCommandResponse().GetStatus())
	assert.Equal(s.T(), "Successfully updated all files", auxResponses[0].GetCommandResponse().GetMessage())
}

//func (s *AuxiliaryTestSuite) TestAuxiliary_ConfigApply() {
//	s.instanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
//	// Perform config apply
//	// Check new config is in both Mocks
//	// Check using hash with new API endpoint which was added to get the file overview
//	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
//	utils.ClearManagementPlaneResponses(s.T(), utils.AuxiliaryMockManagementPlaneAPIAddress)
//
//	ctx := context.Background()
//
//	newConfigFile := "../../config/nginx/nginx-with-test-location.conf"
//
//	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
//		newConfigFile = "../../config/nginx/nginx-plus-with-test-location.conf"
//	}
//
//	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
//		ctx,
//		newConfigFile,
//		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", s.instanceID),
//		0o666,
//	)
//
//	require.NoError(s.T(), err)
//
//	utils.PerformConfigApply(s.T(), s.instanceID, utils.MockManagementPlaneAPIAddress)
//
//	commandResponses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
//
//	sort.Slice(commandResponses, func(i, j int) bool {
//		return commandResponses[i].GetCommandResponse().GetMessage() < commandResponses[j].GetCommandResponse().GetMessage()
//	})
//
//	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, commandResponses[0].GetCommandResponse().GetStatus())
//	assert.Equal(s.T(), "Config apply successful", commandResponses[0].GetCommandResponse().GetMessage())
//	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, commandResponses[1].GetCommandResponse().GetStatus())
//	assert.Equal(s.T(), "Successfully updated all files", commandResponses[1].GetCommandResponse().GetMessage())
//
//	auxResponses := utils.ManagementPlaneResponses(s.T(), 1, utils.AuxiliaryMockManagementPlaneAPIAddress)
//	assert.Equal(s.T(), mpi.CommandResponse_COMMAND_STATUS_OK, auxResponses[1].GetCommandResponse().GetStatus())
//	assert.Equal(s.T(), "Successfully updated all files", auxResponses[1].GetCommandResponse().GetMessage())
//
//	overview := utils.CurrentFileOverview(s.T(), s.instanceID, utils.MockManagementPlaneAPIAddress)
//	overview2 := utils.CurrentFileOverview(s.T(), s.instanceID, utils.AuxiliaryMockManagementPlaneAPIAddress)
//	s.T().Logf("Overview: %v", overview.ConfigVersion)
//	s.T().Logf("Overview 2: %v", overview2.ConfigVersion)
//}

//
//func (s *AuxiliaryTestSuite) TestAuxiliary_ConfigApplyInvalid() {
//	// Perform config apply with aux
//	// Check new config is broken
//	// Check using hash with new API endpoint which was added to get the file overview
//
//}
