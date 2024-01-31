/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConfig_Init(t *testing.T) {
	configPlugin := NewConfig()
	configPlugin.Init(&bus.MessagePipe{})

	assert.NotNil(t, configPlugin.messagePipe)
}

func TestConfig_Info(t *testing.T) {
	configPlugin := NewConfig()
	info := configPlugin.Info()
	assert.Equal(t, "config", info.Name)
}

func TestConfig_Subscriptions(t *testing.T) {
	configPlugin := NewConfig()
	subscriptions := configPlugin.Subscriptions()
	assert.Equal(t, []string{bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC, bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC}, subscriptions)
}

func TestConfig_Process(t *testing.T) {
	testInstance := &instances.Instance{
		InstanceId: "123",
		Type:       instances.Type_NGINX,
	}

	nginxConfigContext := model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.log"}},
		ErrorLogs:  []*model.ErrorLog{{Name: "error.log"}},
	}

	instanceConfigUpdateRequest := &model.InstanceConfigUpdateRequest{
		Instance:      testInstance,
		Location:      "http://file-server.com",
		CorrelationId: "456",
	}

	configurationStatus := &instances.ConfigurationStatus{
		InstanceId:    testInstance.InstanceId,
		CorrelationId: "456",
		Status:        instances.Status_SUCCESS,
		Message:       "Successfully updated instance configuration.",
		LateUpdated:   timestamppb.Now(),
	}

	tests := []struct {
		name     string
		input    *bus.Message
		expected []*bus.Message
	}{
		{
			name: "Instance config updated",
			input: &bus.Message{
				Topic: bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC,
				Data:  configurationStatus,
			},
			expected: []*bus.Message{
				{
					Topic: bus.INSTANCE_CONFIG_CONTEXT_TOPIC,
					Data:  nginxConfigContext,
				},
			},
		},
		{
			name: "Instance config updated - unknown message type",
			input: &bus.Message{
				Topic: bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC,
				Data:  nil,
			},
			expected: nil,
		},
		{
			name: "Instance config update request",
			input: &bus.Message{
				Topic: bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC,
				Data:  instanceConfigUpdateRequest,
			},
			expected: []*bus.Message{
				{
					Topic: bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC,
					Data:  configurationStatus,
				},
			},
		},
		{
			name: "Instance config update request - unknown message type",
			input: &bus.Message{
				Topic: bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC,
				Data:  nil,
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe(context.TODO())
			configPlugin := NewConfig()

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			assert.NoError(tt, err)
			messagePipe.Run()

			configService := &servicefakes.FakeConfigServiceInterface{}
			configService.ParseInstanceConfigurationReturns(nginxConfigContext, nil)
			configService.UpdateInstanceConfigurationReturns(configurationStatus)

			instanceService := &servicefakes.FakeInstanceServiceInterface{}
			instanceService.GetInstanceReturns(testInstance)

			configPlugin.configServices["123"] = configService
			configPlugin.instanceService = instanceService

			configPlugin.Process(test.input)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, test.expected, messages)
		})
	}
}
