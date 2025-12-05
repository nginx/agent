// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/stretchr/testify/suite"
)

const (
	configApplyErrorMessage = "failed to parse config invalid " +
		"number of arguments in \"worker_processes\" directive in /etc/nginx/nginx.conf:1"
)

type ConfigApplyTestSuite struct {
	suite.Suite
	ctx                     context.Context
	teardownTest            func(testing.TB)
	nginxInstanceID         string
	mockManagementConfigDir string
}

type ConfigApplyChunkingTestSuite struct {
	suite.Suite
	ctx             context.Context
	teardownTest    func(testing.TB)
	nginxInstanceID string
}

func (s *ConfigApplyTestSuite) SetupSuite() {
	slog.Info("starting config apply tests")
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), false, false, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)

	s.mockManagementConfigDir = "/mock-management-plane-grpc/config/" + s.nginxInstanceID

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Require().Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Require().Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyTestSuite) TearDownSuite() {
	slog.Info("finished config apply tests")
	s.teardownTest(s.T())
}

func (s *ConfigApplyTestSuite) TearDownTest() {
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
}

// Config Apply with no changes to config
func (s *ConfigApplyTestSuite) TestConfigApply_Test1_TestNoConfigChanges() {
	slog.Info("starting config apply no config changes test")
	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/mime.types": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/mime.types",
				Hash:       "b5XR19dePAcpB9hFYipp0jEQ0SZsFv8SKzEJuLIfOuk=",
				Size:       5349,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "gJ1slpIAUmHAiSo5ZIalKvE40b1hJCgaXasQOMab6kc=",
				Size:       1172,
				Referenced: true,
			},
		},
	}

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Hash = "/SWXYYenb2EcJNg6fiuzlkdj91nBdsMdF1vLm7Wybvc="
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Size = 1218
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful, no files to change", responses[1].GetCommandResponse().GetMessage())
	slog.Info("finished config apply no config changes test")
}

// Config apply -  Add, Update and Delete Referenced file from Management Plane
func (s *ConfigApplyTestSuite) TestConfigApply_Test2_TestValidConfig() {
	slog.Info("starting config apply valid config test")
	// Update nginx.conf
	utils.WriteConfigFileMock(s.T(), s.nginxInstanceID, "/etc/nginx/test/test.conf",
		"/etc/nginx/test/test.conf", "/etc/nginx/test/test.conf")

	// Delete mime.types
	code, _, removeErr := utils.MockManagementPlaneGrpcContainer.Exec(context.Background(), []string{
		"rm",
		s.mockManagementConfigDir + "/etc/nginx/mime.types",
	})

	s.Require().NoError(removeErr)
	s.Equal(0, code)

	// Add test.conf
	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"configs/test.conf",
		s.mockManagementConfigDir+"/etc/nginx/test/test.conf",
		0o666,
	)
	s.Require().NoError(err)

	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/test/test.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/test.conf",
				Hash:       "BF1ztX59kP/N81XcIv3JlPp82j7gzTsVIk2RGxdAta8=",
				Size:       175,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "/SsQwpZTdJVRa1+bex7OdZoogvVT0tnTOwwO59vpsoM=",
				Size:       1360,
				Referenced: true,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].GetCommandResponse().GetMessage() < responses[j].GetCommandResponse().GetMessage()
	})

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
	slog.Info("finished config apply valid config test")
}

// Add, Update and Delete file on DataPlane - Trigger update file overview
func (s *ConfigApplyTestSuite) TestConfigApply_Test3_DataPlaneUpdate() {
	slog.Info("starting config apply data plane update test")
	// Add test2.conf to dataplane
	err := utils.Container.CopyFileToContainer(
		s.ctx,
		"configs/test2.conf",
		"/etc/nginx/test/test2.conf",
		0o666,
	)
	s.Require().NoError(err)

	// Delete test.conf from dataplane
	code, _, removeErr := utils.Container.Exec(context.Background(), []string{
		"rm",
		"/etc/nginx/test/test.conf",
	})

	s.Require().NoError(removeErr)
	s.Equal(0, code)

	// Update nginx.conf to reference new file
	utils.WriteConfigFileDataplane(s.T(), "/etc/nginx/test/test2.conf",
		"/etc/nginx/test/test2.conf", "/etc/nginx/test/test2.conf")

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/test/test2.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/test2.conf",
				Hash:       "mV4nVTx8BObqxSwcJprkJesiCJH+oTO89RgZxFuFEJo=",
				Size:       136,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "q8Zf3Cv5UOAVyfigx5Mr4mwJpLIxApN1H0UzYKKTAiU=",
				Size:       1363,
				Referenced: true,
			},
		},
	}

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
	slog.Info("finished config apply data plane update test")
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test4_TestInvalidConfig() {
	slog.Info("starting config apply invalid config test")
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

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/test/test2.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/test2.conf",
				Hash:       "mV4nVTx8BObqxSwcJprkJesiCJH+oTO89RgZxFuFEJo=",
				Size:       136,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "q8Zf3Cv5UOAVyfigx5Mr4mwJpLIxApN1H0UzYKKTAiU=",
				Size:       1363,
				Referenced: true,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_ERROR, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed, rolling back config", responses[0].GetCommandResponse().GetMessage())
	s.Equal(configApplyErrorMessage, responses[0].GetCommandResponse().GetError())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed, rollback successful", responses[1].GetCommandResponse().GetMessage())
	s.Equal(configApplyErrorMessage, responses[1].GetCommandResponse().GetError())
	slog.Info("finished config apply invalid config test")
}

func (s *ConfigApplyTestSuite) TestConfigApply_Test5_TestFileNotInAllowedDirectory() {
	slog.Info("starting config apply file not in allowed directory test")
	utils.PerformInvalidConfigApply(s.T(), s.nginxInstanceID)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/test/test2.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/test2.conf",
				Hash:       "mV4nVTx8BObqxSwcJprkJesiCJH+oTO89RgZxFuFEJo=",
				Size:       136,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "q8Zf3Cv5UOAVyfigx5Mr4mwJpLIxApN1H0UzYKKTAiU=",
				Size:       1363,
				Referenced: true,
			},
		},
	}
	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_FAILURE, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply failed", responses[0].GetCommandResponse().GetMessage())
	s.Equal(
		"file not in allowed directories /unknown/nginx.conf",
		responses[0].GetCommandResponse().GetError(),
	)
	slog.Info("finished config apply file not in allowed directory test")
}

func (s *ConfigApplyChunkingTestSuite) SetupSuite() {
	slog.Info("starting config apply chunking tests")
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), false, false, false,
		"../../config/agent/nginx-config-with-max-file-size.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Require().Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Require().Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyChunkingTestSuite) TearDownSuite() {
	slog.Info("finished config apply chunking tests")
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

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/mime.types": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/mime.types",
				Hash:       "b5XR19dePAcpB9hFYipp0jEQ0SZsFv8SKzEJuLIfOuk=",
				Size:       5349,
				Referenced: true,
			},
		},
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "dfDpjGOjOhWWhX43y/d+zBulXCisx+BVYj2eEEud6ac=",
				Size:       886910,
				Referenced: true,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
}

func TestConfigApplyTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigApplyTestSuite))
	suite.Run(t, new(ConfigApplyChunkingTestSuite))
	suite.Run(t, new(ConfigApplyUnreferencedFilesTestSuite))
}
