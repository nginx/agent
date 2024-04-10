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

	"github.com/nginx/agent/v3/internal/client/clientfakes"

	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/internal/config"
	configfakes2 "github.com/nginx/agent/v3/internal/datasource/config/configfakes"
	"github.com/nginx/agent/v3/internal/service/config/configfakes"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const instanceID = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"

func TestConfigService_SetConfigContext(t *testing.T) {
	ctx := context.Background()

	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	instance := &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   instanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
	}

	configService := NewConfigService(ctx, instance, &config.Config{
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	}, &clientfakes.FakeConfigClient{})
	configService.SetConfigContext(expectedConfigContext)

	assert.Equal(t, expectedConfigContext, configService.configContext)
}

func TestUpdateInstanceConfiguration(t *testing.T) {
	ctx := context.Background()
	instance := &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   instanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
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
			name:        "Test 1: Write fails",
			writeErr:    fmt.Errorf("error writing config"),
			validateErr: nil,
			applyErr:    nil,
			completeErr: nil,
			expected:    protos.CreateFailStatus("error writing config"),
		},
		{
			name:        "Test 2: Validate fails",
			writeErr:    nil,
			validateErr: fmt.Errorf("error validating config"),
			applyErr:    nil,
			completeErr: nil,
			expected:    protos.CreateFailStatus("error validating config"),
		},
		{
			name:        "Test 3: Apply fails",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    fmt.Errorf("error reloading config"),
			completeErr: nil,
			expected:    protos.CreateFailStatus("error reloading config"),
		},
		{
			name:        "Test 4: Complete fails",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    nil,
			completeErr: fmt.Errorf("error completing config apply"),
			expected:    protos.CreateSuccessStatus(),
		},
		{
			name:        "Test 5: Successfully updated config",
			writeErr:    nil,
			validateErr: nil,
			applyErr:    nil,
			expected:    protos.CreateSuccessStatus(),
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

			cs := NewConfigService(ctx, instance, agentConfig, &clientfakes.FakeConfigClient{})
			cs.configService = &mockService
			_, result := cs.UpdateInstanceConfiguration(ctx, filesURL)

			assert.Equal(t, test.expected.GetStatus(), result.GetStatus())
			assert.Equal(t, test.expected.GetMessage(), result.GetMessage())
			assert.Equal(t, test.expected.GetInstanceId(), result.GetInstanceId())
		})
	}
}

func TestConfigService_ParseInstanceConfiguration(t *testing.T) {
	ctx := context.Background()

	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
	}

	instance := &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   instanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
	}

	configService := NewConfigService(ctx, instance, &config.Config{
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	}, &clientfakes.FakeConfigClient{})

	fakeDataPlaneConfig := &configfakes.FakeDataPlaneConfig{}
	fakeDataPlaneConfig.ParseConfigReturns(expectedConfigContext, nil)

	configService.configService = fakeDataPlaneConfig

	result, err := configService.ParseInstanceConfiguration(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
