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

type ConfigApplyUnreferencedFilesTestSuite struct {
	suite.Suite
	ctx                     context.Context
	teardownTest            func(testing.TB)
	nginxInstanceID         string
	mockManagementConfigDir string
}

func (s *ConfigApplyUnreferencedFilesTestSuite) SetupSuite() {
	slog.Info("starting config apply with unreferenced files tests")
	s.ctx = context.Background()
	s.teardownTest = utils.SetupConnectionTest(s.T(), false, false, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	s.nginxInstanceID = utils.VerifyConnection(s.T(), 2, utils.MockManagementPlaneAPIAddress)

	s.mockManagementConfigDir = "/mock-management-plane-grpc/config/" + s.nginxInstanceID

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Require().Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Require().Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())
}

func (s *ConfigApplyUnreferencedFilesTestSuite) TearDownSuite() {
	slog.Info("finished config apply with unreferenced files tests")
	s.teardownTest(s.T())
}

func (s *ConfigApplyUnreferencedFilesTestSuite) TearDownTest() {
	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
}

// Config apply with unreferenced file in subdirectory
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test1_TestSubDirectory() {
	slog.Info("starting config apply unreferenced file in sub directory test")

	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"configs/unreferenced_file.conf",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/test/unreferenced_file.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	body := `{
			"unreferencedFiles": [
				{
					"file_meta": {
						"name": "/etc/nginx/test/unreferenced_file.conf",
						"permissions": "0644"
					}
				}
			]
		}`

	utils.PerformConfigApplyWithRequestBody(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress, body)
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
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
			},
		},
	}

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Hash = "/SWXYYenb2EcJNg6fiuzlkdj91nBdsMdF1vLm7Wybvc="
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Size = 1218
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	sort.Slice(responses, func(i, j int) bool {
		return responses[i].GetCommandResponse().GetMessage() < responses[j].GetCommandResponse().GetMessage()
	})

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
	slog.Info("finished config apply unreferenced file in sub directory test")
}

// Config apply to update unreferenced file in DataPlane
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test2_TestUpdateUnreferencedInDataPlane() {
	slog.Info("starting update unreferenced file in data plane test")

	originalContent, readErr := os.ReadFile("configs/unreferenced_file.conf")
	s.Require().NoError(readErr)

	code, _, updateErr := utils.Container.Exec(s.ctx, []string{
		"sh", "-c", "echo '# Updated unreferenced file' >> /etc/nginx/test/unreferenced_file.conf",
	})

	s.Require().NoError(updateErr)
	s.Equal(0, code)

	body := `{
			"unreferencedFiles": [
				{
					"file_meta": {
						"name": "/etc/nginx/test/unreferenced_file.conf",
						"permissions": "0644"
					}
				}
			]
		}`

	utils.PerformConfigApplyWithRequestBody(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress, body)

	code, output, outputErr := utils.Container.Exec(s.ctx, []string{
		"cat", "/etc/nginx/test/unreferenced_file.conf",
	})
	s.Require().NoError(outputErr)
	s.Equal(0, code)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
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
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
			},
		},
	}

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Hash = "/SWXYYenb2EcJNg6fiuzlkdj91nBdsMdF1vLm7Wybvc="
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Size = 1218
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful, no files to change", responses[0].GetCommandResponse().GetMessage())
	s.NotEqual(originalContent, output)
	slog.Info("finished update unreferenced file in data plane test")
}

// Config apply to delete unreferenced file from DataPlane
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test3_TestDeleteUnreferencedInDataPlane() {
	slog.Info("starting delete unreferenced file from data plane test")

	code, _, removeErr := utils.Container.Exec(context.Background(), []string{
		"rm",
		"/etc/nginx/test/unreferenced_file.conf",
	})

	s.Require().NoError(removeErr)
	s.Equal(0, code)

	code, _, err := utils.Container.Exec(s.ctx, []string{
		"test", "-f", "/etc/nginx/test/unreferenced_file.conf",
	})
	s.Require().NoError(err)
	s.NotEqual(0, code)

	body := `{
			"unreferencedFiles": [
				{
					"file_meta": {
						"name": "/etc/nginx/test/unreferenced_file.conf",
						"permissions": "0644"
					}
				}
			]
		}`

	utils.PerformConfigApplyWithRequestBody(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress, body)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
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
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
			},
		},
	}

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Hash = "/SWXYYenb2EcJNg6fiuzlkdj91nBdsMdF1vLm7Wybvc="
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Size = 1218
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())

	code, _, err = utils.Container.Exec(s.ctx, []string{
		"test", "-f", "/etc/nginx/test/unreferenced_file.conf",
	})
	s.Require().NoError(err)
	s.Equal(0, code)
	slog.Info("finished delete unreferenced file from data plane test")
}

// Config apply to delete unreferenced file from Mock
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test4_TestDeleteUnreferencedFromMock() {
	slog.Info("starting delete unreferenced file from mock test")

	code, _, removeErr := utils.MockManagementPlaneGrpcContainer.Exec(context.Background(), []string{
		"rm",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/test/unreferenced_file.conf", s.nginxInstanceID),
	})

	s.Require().NoError(removeErr)
	s.Equal(0, code)

	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
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
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())

	code, _, err := utils.MockManagementPlaneGrpcContainer.Exec(s.ctx, []string{
		"test",
		"-f",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/test/unreferenced_file.conf", s.nginxInstanceID),
	})
	s.Require().NoError(err)
	s.NotEqual(0, code)

	code, _, err = utils.Container.Exec(s.ctx, []string{
		"test", "-f", "/etc/nginx/test/unreferenced_file.conf",
	})
	s.Require().NoError(err)
	s.NotEqual(0, code)

	slog.Info("finished delete unreferenced file from mock test")
}

// Config apply to change unreferenced file to referenced file
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test5_TestUnreferencedToReferenced() {
	slog.Info("starting unreferenced file to referenced file test")

	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"configs/unreferenced_file.conf",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/test/unreferenced_file.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	body := `{
			"unreferencedFiles": [
				{
					"file_meta": {
						"name": "/etc/nginx/test/unreferenced_file.conf",
						"permissions": "0644"
					}
				}
			]
		}`

	utils.PerformConfigApplyWithRequestBody(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress, body)
	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.T().Logf("Config apply responses: %v", responses)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())

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
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
			},
		},
	}

	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Hash = "/SWXYYenb2EcJNg6fiuzlkdj91nBdsMdF1vLm7Wybvc="
		manifestFiles["/etc/nginx/nginx.conf"].ManifestFileMeta.Size = 1218
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)
	utils.WriteConfigFileMock(s.T(), s.nginxInstanceID, "/etc/nginx/test/unreferenced_file.conf",
		"/etc/nginx/test/unreferenced_file.conf", "/etc/nginx/mime.types")

	utils.ClearManagementPlaneResponses(s.T(), utils.MockManagementPlaneAPIAddress)
	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses = utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)

	manifestFiles = map[string]*model.ManifestFile{
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
				Hash:       "NIs0JY8C/mhUGfarLe28m3oQmeqc8+4MKXzLtWAgwGI=",
				Size:       1382,
				Referenced: true,
			},
		},
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: true,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
	slog.Info("finished unreferenced file to referenced file test")
}

// Config apply to change referenced file to unreferenced file
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test6_TestReferencedToUnreferenced() {
	slog.Info("starting referenced file to unreferenced file test")

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
				Hash:       "NIs0JY8C/mhUGfarLe28m3oQmeqc8+4MKXzLtWAgwGI=",
				Size:       1382,
				Referenced: true,
			},
		},
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: true,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)
	utils.WriteConfigFileMock(s.T(), s.nginxInstanceID, "/etc/nginx/mime.types",
		"/etc/nginx/mime.types", "/etc/nginx/mime.types")

	utils.PerformConfigApply(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress)
	responses := utils.ManagementPlaneResponses(s.T(), 2, utils.MockManagementPlaneAPIAddress)

	manifestFiles = map[string]*model.ManifestFile{
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
				Hash:       "r+khc9eBiffYMXGIdkQ3CeGar4/MBzuMUkaSlcSXsOw=",
				Size:       1348,
				Referenced: true,
			},
		},
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
			},
		},
	}

	utils.CheckManifestFile(s.T(), utils.Container, manifestFiles)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Config apply successful", responses[0].GetCommandResponse().GetMessage())
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[1].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[1].GetCommandResponse().GetMessage())
	slog.Info("finished referenced file to unreferenced file test")
}

// Config apply with invalid config and unreferenced file to test rollback
func (s *ConfigApplyUnreferencedFilesTestSuite) TestConfigApply_Test7_TestRollbackWithUnreferencedFile() {
	slog.Info("starting invalid config apply with unreferenced file test")

	err := utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"../../config/nginx/invalid-nginx.conf",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/nginx.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	err = utils.MockManagementPlaneGrpcContainer.CopyFileToContainer(
		s.ctx,
		"./configs/unreferenced_file.conf",
		fmt.Sprintf("/mock-management-plane-grpc/config/%s/etc/nginx/unreferenced_file.conf", s.nginxInstanceID),
		0o666,
	)
	s.Require().NoError(err)

	body := `{
			"unreferencedFiles": [
				{
					"file_meta": {
						"name": "/etc/nginx/unreferenced_file.conf",
						"permissions": "0644"
					}
				}
			]
		}`

	utils.PerformConfigApplyWithRequestBody(s.T(), s.nginxInstanceID, utils.MockManagementPlaneAPIAddress, body)
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
				Hash:       "r+khc9eBiffYMXGIdkQ3CeGar4/MBzuMUkaSlcSXsOw=",
				Size:       1348,
				Referenced: true,
			},
		},
		"/etc/nginx/test/unreferenced_file.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/test/unreferenced_file.conf",
				Hash:       "ucNsmG0hN5ojrMVkQKveSGlt00uIaEkZ1rTDa1QNUY0=",
				Size:       189,
				Referenced: false,
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
	slog.Info("finished invalid config apply with unreferenced file test")
}
