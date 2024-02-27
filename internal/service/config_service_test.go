// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/nginx/agent/v3/internal/config"
	configfakes2 "github.com/nginx/agent/v3/internal/datasource/config/configfakes"
	"github.com/nginx/agent/v3/internal/service/config/configfakes"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const instanceID = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

func TestConfigService_SetConfigContext(t *testing.T) {
	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	instance := &instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}

	configService := NewConfigService(instance, &config.Config{})
	configService.SetConfigContext(expectedConfigContext)

	assert.Equal(t, expectedConfigContext, configService.configContext)
}

func TestUpdateInstanceConfiguration(t *testing.T) {
	instanceID := "ae6c58c1-bc92-30c1-a9c9-85591422068e"
	correlationID := "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	ctx := context.TODO()
	instance := instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}
	agentConfig := config.Config{}

	tests := []struct {
		name        string
		writeErr    error
		validateErr error
		reloadErr   error
		expected    *instances.ConfigurationStatus
	}{
		{
			name:        "write fails",
			writeErr:    fmt.Errorf("error writing config"),
			validateErr: nil,
			reloadErr:   nil,
			expected: &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_FAILED,
				Message:       "error writing config",
			},
		},
		{
			name:        "validate fails",
			writeErr:    nil,
			validateErr: fmt.Errorf("error validating config"),
			reloadErr:   nil,
			expected: &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_FAILED,
				Message:       "error validating config",
			},
		},
		{
			name:        "reload fails",
			writeErr:    nil,
			validateErr: nil,
			reloadErr:   fmt.Errorf("error reloading config"),
			expected: &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_FAILED,
				Message:       "error reloading config",
			},
		},
		{
			name:        "success",
			writeErr:    nil,
			validateErr: nil,
			reloadErr:   nil,
			expected: &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_SUCCESS,
				Message:       "Config applied successfully",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := configfakes.FakeDataPlaneConfig{}
			mockConfigWriter := configfakes2.FakeConfigWriterInterface{}

			mockService.SetConfigWriter(&mockConfigWriter)
			mockConfigWriter.WriteReturns(nil, test.writeErr)
			mockService.WriteReturns(nil, test.writeErr)
			mockService.ApplyReturns(test.reloadErr)
			mockService.ValidateReturns(test.validateErr)

			filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

			cs := NewConfigService(&instance, &agentConfig)
			cs.configService = &mockService
			result := cs.UpdateInstanceConfiguration(ctx, correlationID, filesURL)
			assert.Equal(t, test.expected, result)
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

	configService := NewConfigService(instance, &config.Config{})

	fakeDataPlaneConfig := &configfakes.FakeDataPlaneConfig{}
	fakeDataPlaneConfig.ParseConfigReturns(expectedConfigContext, nil)

	configService.configService = fakeDataPlaneConfig

	result, err := configService.ParseInstanceConfiguration("123")

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
