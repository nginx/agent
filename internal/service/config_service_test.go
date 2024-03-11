// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/internal/config"
	configfakes2 "github.com/nginx/agent/v3/internal/datasource/config/configfakes"
	"github.com/nginx/agent/v3/internal/service/config/configfakes"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const instanceID = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"

func TestConfigService_SetConfigContext(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	instance := &instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}

	configService := NewConfigService(instance, &config.Config{
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	})
	configService.SetConfigContext(expectedConfigContext)

	assert.Equal(t, expectedConfigContext, configService.configContext)
}

func TestUpdateInstanceConfiguration(t *testing.T) {
	correlationID := "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	ctx := context.TODO()
	instance := instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}
	agentConfig := types.GetAgentConfig()

	tests := []struct {
		name        string
		writeErr    error
		validateErr error
		applyErr    error
		completeErr error
		expected    *instances.ConfigurationStatus
	}{
		{
			name:        "write fails",
			writeErr:    fmt.Errorf("error writing config"),
			validateErr: nil,
			applyErr:    nil,
			completeErr: nil,
			expected:    helpers.CreateFailStatus("error writing config"),
		},
		{
			name:        "validate fails",
			writeErr:    nil,
			validateErr: fmt.Errorf("error validating config"),
			applyErr:    nil,
			completeErr: nil,
			expected:    helpers.CreateFailStatus("error validating config"),
		},
		{
			name:        "apply fails",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    fmt.Errorf("error reloading config"),
			completeErr: nil,
			expected:    helpers.CreateFailStatus("error reloading config"),
		},
		{
			name:        "complete fails",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    nil,
			completeErr: fmt.Errorf("error completing config apply"),
			expected:    helpers.CreateSuccessStatus(),
		},
		{
			name:        "success",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    nil,
			expected:    helpers.CreateSuccessStatus(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := configfakes.FakeDataPlaneConfig{}
			mockConfigWriter := configfakes2.FakeConfigWriterInterface{}

			mockService.SetConfigWriter(&mockConfigWriter)
			mockConfigWriter.WriteReturns(nil, test.writeErr)
			mockService.WriteReturns(nil, test.writeErr)
			mockService.ApplyReturns(test.applyErr)
			mockService.ValidateReturns(test.validateErr)
			mockService.CompleteReturns(test.completeErr)

			filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

			cs := NewConfigService(&instance, agentConfig)
			cs.configService = &mockService
			_, result := cs.UpdateInstanceConfiguration(ctx, correlationID, filesURL)

			assert.Equal(t, test.expected.GetStatus(), result.GetStatus())
			assert.Equal(t, test.expected.GetMessage(), result.GetMessage())
			assert.Equal(t, test.expected.GetInstanceId(), result.GetInstanceId())
		})
	}
}

func TestConfigService_ParseInstanceConfiguration(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	instance := &instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}

	configService := NewConfigService(instance, &config.Config{
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	})

	fakeDataPlaneConfig := &configfakes.FakeDataPlaneConfig{}
	fakeDataPlaneConfig.ParseConfigReturns(expectedConfigContext, nil)

	configService.configService = fakeDataPlaneConfig

	result, err := configService.ParseInstanceConfiguration("123")

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
