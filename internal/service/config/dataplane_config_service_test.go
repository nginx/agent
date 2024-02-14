// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

type FakeDataplaneConfig struct{}

// nolint: unparam // always returns nil but is a test
func (*FakeDataplaneConfig) ParseConfig(_ *instances.Instance) (any, error) {
	return &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}, nil
}

func TestConfig_ParseConfig(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	fakeDataplaneConfig := &FakeDataplaneConfig{}
	result, err := fakeDataplaneConfig.ParseConfig(&instances.Instance{Type: instances.Type_NGINX})

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
