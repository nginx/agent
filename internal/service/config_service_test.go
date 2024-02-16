// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"testing"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/config/configfakes"
	"github.com/stretchr/testify/assert"
)

const instanceID = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

func TestConfigService_SetConfigContext(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	configService := NewConfigService(instanceID, &config.Config{})
	configService.SetConfigContext(expectedConfigContext)

	assert.Equal(t, expectedConfigContext, configService.configContext)
}

func TestConfigService_ParseInstanceConfiguration(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	configService := NewConfigService(instanceID, &config.Config{})

	fakeDataplaneConfig := &configfakes.FakeDataplaneConfig{}
	fakeDataplaneConfig.ParseConfigReturns(expectedConfigContext, nil)

	configService.dataplaneConfigServices[instances.Type_NGINX] = fakeDataplaneConfig

	result, err := configService.ParseInstanceConfiguration("123", &instances.Instance{Type: instances.Type_NGINX})

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)

	_, err = configService.ParseInstanceConfiguration("123", &instances.Instance{Type: instances.Type_UNKNOWN})

	require.Error(t, err)
}
