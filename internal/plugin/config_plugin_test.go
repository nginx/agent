// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/internal/service/servicefakes"
	modelHelpers "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
)

const (
	correlationID = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
)

func TestConfig_Init(t *testing.T) {
	ctx := context.Background()
	configPlugin := NewConfig(&config.Config{})
	err := configPlugin.Init(ctx, &bus.MessagePipe{})
	require.NoError(t, err)

	assert.NotNil(t, configPlugin.messagePipe)
}

func TestConfig_Info(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	info := configPlugin.Info()
	assert.Equal(t, "config", info.Name)
}

func TestConfig_Close(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	err := configPlugin.Close(context.Background())
	require.NoError(t, err)
}

func TestConfig_Subscriptions(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	subscriptions := configPlugin.Subscriptions()
	assert.Equal(t, []string{
		bus.InstanceConfigUpdateRequestTopic,
		bus.InstanceConfigUpdateStatusTopic,
		bus.ConfigClientTopic,
		bus.ResourceTopic,
	}, subscriptions)
}

func TestConfig_Process(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		logger.CorrelationIDContextKey,
		slog.Any(logger.CorrelationIDKey, correlationID),
	)

	testInstance := protos.GetNginxOssInstance()

	nginxConfigContext := modelHelpers.GetConfigContext()

	instanceConfigUpdateRequest := &v1.ManagementPlaneRequest_ConfigApplyRequest{
		ConfigApplyRequest: &v1.ConfigApplyRequest{
			ConfigVersion: &v1.ConfigVersion{
				Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
				InstanceId: testInstance.GetInstanceMeta().GetInstanceId(),
			},
		},
	}

	configurationStatusProgress := protos.CreateInProgressStatus()
	configurationStatus := protos.CreateSuccessStatus()

	tests := []struct {
		name     string
		input    *bus.Message
		expected []*bus.Message
	}{
		{
			name: "Test 1: Instance config updated",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateStatusTopic,
				Data:  configurationStatus,
			},
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigContextTopic,
					Data:  nginxConfigContext,
				},
			},
		},
		{
			name: "Test 2: Instance config updated - unknown message type",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateStatusTopic,
				Data:  nil,
			},
			expected: nil,
		},
		{
			name: "Test 3: Instance config update request",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateRequestTopic,
				Data:  instanceConfigUpdateRequest,
			},
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  configurationStatusProgress,
				},
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  protos.CreateSuccessStatus(),
				},
			},
		},
		{
			name: "Test 4: Instance config update request - unknown message type",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateRequestTopic,
				Data:  nil,
			},
			expected: nil,
		},
		{
			name: "Test 5: Resource request",
			input: &bus.Message{
				Topic: bus.ResourceTopic,
				Data:  protos.GetContainerizedResource(),
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe()
			configPlugin := NewConfig(&config.Config{
				Client: &config.Client{Timeout: 1 * time.Second},
			})

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			require.NoError(tt, err)
			messagePipe.Run(ctx)

			configService := &servicefakes.FakeConfigServiceInterface{}
			configService.ParseInstanceConfigurationReturns(nginxConfigContext, nil)
			configService.UpdateInstanceConfigurationReturns(nil, configurationStatus)

			configPlugin.configServices[protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId()] = configService
			configPlugin.resource.Instances = []*v1.Instance{testInstance}

			configPlugin.Process(ctx, test.input)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, len(test.expected), len(messages))

			for key, message := range test.expected {
				assert.Equal(tt, message.Topic, messages[key].Topic)
			}
		})
	}
}

func TestConfig_Update(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		logger.CorrelationIDContextKey,
		slog.Any(logger.CorrelationIDKey, correlationID),
	)

	agentConfig := types.GetAgentConfig()
	instance := protos.GetNginxOssInstance()

	request := &v1.ManagementPlaneRequest_ConfigApplyRequest{
		ConfigApplyRequest: &v1.ConfigApplyRequest{
			ConfigVersion: &v1.ConfigVersion{
				Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
				InstanceId: protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId(),
			},
		},
	}

	inProgressStatus := protos.CreateInProgressStatus()
	successStatus := protos.CreateSuccessStatus()
	failStatus := protos.CreateFailStatus("error")
	// rollbackInProgressStatus := protos.CreateRollbackInProgressStatus()

	tests := []struct {
		name               string
		updateReturnStatus *instances.ConfigurationStatus
		rollbackReturns    error
		expected           []*bus.Message
	}{
		{
			name:               "Test 1: Successful config update",
			updateReturnStatus: successStatus,
			rollbackReturns:    nil,
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  successStatus,
				},
			},
		},
		{
			// removed rollback part of this test for now
			name:               "Test 2: Config update failed and rolled back",
			updateReturnStatus: failStatus,
			rollbackReturns:    nil,
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  failStatus,
				},
			},
		},
		{
			// removed rollback part of this test for now
			name:               "Test 2: Rollback fails",
			updateReturnStatus: failStatus,
			rollbackReturns:    fmt.Errorf("rollback failed"),
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateStatusTopic,
					Data:  failStatus,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe()
			configPlugin := NewConfig(agentConfig)

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			require.NoError(tt, err)
			messagePipe.Run(ctx)

			configService := &servicefakes.FakeConfigServiceInterface{}
			configService.UpdateInstanceConfigurationReturns(make(map[string]*v1.FileMeta), test.updateReturnStatus)
			configService.RollbackReturns(test.rollbackReturns)

			instanceService := []*v1.Instance{instance}
			configPlugin.configServices[protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId()] = configService
			configPlugin.resource.Instances = instanceService

			configPlugin.updateInstanceConfig(ctx, request)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, len(test.expected), len(messages))

			for key, message := range test.expected {
				assert.Equal(tt, message.Topic, messages[key].Topic)
			}
		})
	}
}
