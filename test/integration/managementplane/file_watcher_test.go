// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package managementplane

import (
	"github.com/nginx/agent/v3/test/integration/utils"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

func (s *MPITestSuite) TestFileWatcher_Test1_TestUpdateNGINXConfig() {
	err := utils.Container.CopyFileToContainer(
		s.ctx,
		"../../config/nginx/nginx-with-server-block-access-log.conf",
		"/etc/nginx/nginx.conf",
		0o666,
	)
	s.Require().NoError(err)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)

	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	utils.VerifyUpdateDataPlaneStatus(s.T(), utils.MockManagementPlaneAPIAddress)
}

func (s *MPITestSuite) TestFileWatcher_Test2_TestCreateNGINXConfig() {
	err := utils.Container.CopyFileToContainer(
		s.ctx,
		"../../config/nginx/empty-nginx.conf",
		"/etc/nginx/test/test.conf",
		0o666,
	)
	s.Require().NoError(err)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	utils.VerifyUpdateDataPlaneStatus(s.T(), utils.MockManagementPlaneAPIAddress)
}

func (s *MPITestSuite) TestFileWatcher_Test3_TestDeleteNGINXConfig() {
	_, _, err := utils.Container.Exec(
		s.ctx,
		[]string{"rm", "-rf", "/etc/nginx/test"},
	)
	s.Require().NoError(err)

	responses := utils.ManagementPlaneResponses(s.T(), 1, utils.MockManagementPlaneAPIAddress)
	s.Equal(mpi.CommandResponse_COMMAND_STATUS_OK, responses[0].GetCommandResponse().GetStatus())
	s.Equal("Successfully updated all files", responses[0].GetCommandResponse().GetMessage())

	utils.VerifyUpdateDataPlaneStatus(s.T(), utils.MockManagementPlaneAPIAddress)
}
