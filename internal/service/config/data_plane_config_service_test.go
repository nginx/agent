// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

type FakeDataPlaneConfig struct{}

// nolint: unparam // always returns nil but is a test
func (*FakeDataPlaneConfig) ParseConfig(_ *v1.Instance) (any, error) {
	return &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}, nil
}

func TestConfig_ParseConfig(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	fakeDataPlaneConfig := &FakeDataPlaneConfig{}
	result, err := fakeDataPlaneConfig.ParseConfig(
		&v1.Instance{
			InstanceMeta: &v1.InstanceMeta{
				InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
