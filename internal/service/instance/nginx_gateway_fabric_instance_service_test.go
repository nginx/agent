// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"testing"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNginxGatewayFabric_GetInstances(t *testing.T) {
	result := NewNginxGatewayFabric().GetInstances([]*model.Process{})
	assert.Empty(t, result)
}
