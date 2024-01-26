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
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

type FakeDataplaneConfig struct{}

func (*FakeDataplaneConfig) ParseConfig(instance *instances.Instance) (any, error) {
	return &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}, nil
}

func TestConfig_ParseConfig(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	fakeDataplaneConfig := &FakeDataplaneConfig{}
	testConfig := &Config[DataplaneConfig]{fakeDataplaneConfig}
	result, err := testConfig.ParseConfig(&instances.Instance{Type: instances.Type_NGINX})

	assert.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
