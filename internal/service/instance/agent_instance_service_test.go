// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
)

func TestNginxAgent_GetInstances(t *testing.T) {
	ctx := context.Background()
	result := NewNginxAgent(types.GetAgentConfig()).GetInstances(ctx, make(map[int32]*model.Process))
	assert.Len(t, result, 1)

	assert.Equal(t, types.GetAgentConfig().UUID, result[0].GetInstanceMeta().GetInstanceId())
	assert.Equal(t, types.GetAgentConfig().Path, result[0].GetInstanceRuntime().GetConfigPath())
	// when populated, adjust this assertion
	assert.Equal(t, "", result[0].GetInstanceConfig().GetAgentConfig().GetCommand().String())
}
