// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"testing"

	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildInfo(t *testing.T) {
	agentConfig := types.AgentConfig()
	info := BuildInfo(agentConfig)

	assert.Equal(t, agentConfig.Version, info.Version)
	assert.NotEmpty(t, info.Description)
	assert.NotNil(t, info.Command)
}
