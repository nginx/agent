/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/stretchr/testify/assert"
)

func TestNginxGatewayFabric_ParseConfig(t *testing.T) {
	result, err := NewNginxGatewayFabric().ParseConfig(&instances.Instance{})
	// Not implemented yet so error is expected
	assert.Nil(t, result)
	assert.Error(t, err)
}
