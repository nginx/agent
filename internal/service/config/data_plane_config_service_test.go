// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	modelHelpers "github.com/nginx/agent/v3/test/model"

	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

type FakeDataPlaneConfig struct{}

// nolint: unparam // always returns nil but is a test
func (*FakeDataPlaneConfig) ParseConfig(_ *v1.Instance) (any, error) {
	return modelHelpers.GetConfigContext(), nil
}

func TestConfig_ParseConfig(t *testing.T) {
	expectedConfigContext := modelHelpers.GetConfigContext()

	fakeDataPlaneConfig := &FakeDataPlaneConfig{}
	result, err := fakeDataPlaneConfig.ParseConfig(
		protos.GetNginxOssInstance([]string{}),
	)

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
