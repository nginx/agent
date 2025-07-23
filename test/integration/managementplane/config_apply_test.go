// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/suite"
)

const (
	configApplyErrorMessage = "failed to parse config invalid " +
		"number of arguments in \"worker_processes\" directive in /etc/nginx/nginx.conf:1"
)

type ConfigApplyTestSuite struct {
	suite.Suite
	ctx             context.Context
	teardownTest    func(testing.TB)
	nginxInstanceID string
}

type ConfigApplyChunkingTestSuite struct {
	suite.Suite
	ctx             context.Context
	teardownTest    func(testing.TB)
	nginxInstanceID string
}

func (s *ConfigApplyTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), false, false, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Require().Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Require().Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyTestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

func (s *ConfigApplyTestSuite) TearDownTest() {
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test1_TestNoConfigChanges() {
	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful, no files to change", responses[1].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test2_TestValidConfig() {
	newConfigFile := "../../config/nginx/nginx-with-test-location.conf"

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		newConfigFile = "../../config/nginx/nginx-plus-with-test-location.conf"
	}
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
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test3_TestInvalidConfig() {
	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"../../config/nginx/invalid-nginx.conf",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)

	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
	s.Equal(configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
	s.Equal(configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test4_TestFileNotInAllowedDirectory() {
	utils.PerformInvalidConfigApply(s.T(), s.nginxInstanceID)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed", responses[0].GetCommandResponse().GetMessage())
	s.Equal(
		"file not in allowed directories /unknown/nginx.conf",
		responses[0].GetCommandResponse().GetError(),
	)
}

func (s *ConfigApplyChunkingTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), false, false, false,
		"../../config/agent/nginx-config-with-max-file-size.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Require().Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Require().Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyChunkingTestSuite) TearDownSuite() {
	s.teardownTest(s.T())
}

func (s *ConfigApplyChunkingTestSuite) TestConfigApplyChunking() {
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)

	newConfigFile := "../../config/nginx/nginx-1mb-file.conf"

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
}

func TestConfigApplyTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigApplyTestSuite))
	suite.Run(t, new(ConfigApplyChunkingTestSuite))
}
