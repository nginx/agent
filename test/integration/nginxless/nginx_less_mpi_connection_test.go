// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginxless

import (
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/stretchr/testify/assert"
)

// Verify that the agent sends a connection request to Management Plane even when Nginx is not present
func TestNginxLessGrpc_Connection(t *testing.T) {
	teardownTest := utils.SetupConnectionTest(t, true, true, false,
		"../../config/agent/nginx-config-with-grpc-client.conf")
	defer teardownTest(t)

	utils.VerifyConnection(t, 1, utils.MockManagementPlaneAPIAddress)
	assert.False(t, t.Failed())
}
