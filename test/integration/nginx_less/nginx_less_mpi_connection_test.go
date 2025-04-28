// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx_less

import (
	"testing"

	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/stretchr/testify/assert"
)

// Verify that the agent sends a connection request to Management Plane even when Nginx is not present
func TestNginxLessGrpc_Connection(t *testing.T) {
	teardownTest := utils.SetupConnectionTest(t, true, true)
	defer teardownTest(t)

	utils.VerifyConnection(t, 1)
	assert.False(t, t.Failed())
}
