// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package auxiliarycommandserver

import (
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"
	"github.com/stretchr/testify/assert"
)

func TestAuxiliary_StartUp(t *testing.T) {
	teardownTest := utils.SetupConnectionTest(t, true, false, true,
		"../../config/agent/nginx-agent-with-auxiliary-command.conf")

	defer teardownTest(t)

	utils.VerifyConnection(t, 2, utils.MockManagementPlaneAPIAddress)
	assert.False(t, t.Failed())
	utils.VerifyUpdateDataPlaneHealth(t, utils.MockManagementPlaneAPIAddress)

	utils.VerifyConnection(t, 2, utils.AuxiliaryMockManagementPlaneAPIAddress)
	assert.False(t, t.Failed())
	utils.VerifyUpdateDataPlaneHealth(t, utils.AuxiliaryMockManagementPlaneAPIAddress)
}
