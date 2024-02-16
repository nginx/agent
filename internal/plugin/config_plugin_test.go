// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConfig_Init(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	configPlugin.Init(&bus.MessagePipe{})

	assert.NotNil(t, configPlugin.messagePipe)
}

func TestConfig_Info(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	info := configPlugin.Info()
	assert.Equal(t, "config", info.Name)
}

func TestConfig_Subscriptions(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	subscriptions := configPlugin.Subscriptions()
	assert.Equal(t, []string{
		bus.InstanceConfigUpdateRequestTopic,
		bus.InstanceConfigUpdateCompleteTopic,
	}, subscriptions)
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
		CorrelationID: "456",
	}

	configurationStatus := &instances.ConfigurationStatus{
		InstanceId:    testInstance.GetInstanceId(),
		CorrelationId: "456",
		Status:        instances.Status_SUCCESS,
		Message:       "Successfully updated instance configuration.",
		LastUpdated:   timestamppb.Now(),
	}

	tests := []struct {
		name     string
		input    *bus.Message
		expected []*bus.Message
	}{
		{
			name: "Instance config updated",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateCompleteTopic,
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
			name: "Instance config updated - unknown message type",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateCompleteTopic,
				Data:  nil,
			},
			expected: nil,
		},
		{
			name: "Instance config update request",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateRequestTopic,
				Data:  instanceConfigUpdateRequest,
			},
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateCompleteTopic,
					Data:  configurationStatus,
				},
			},
		},
		{
			name: "Instance config update request - unknown message type",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateRequestTopic,
				Data:  nil,
			},
			expected: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe(context.TODO())
			configPlugin := NewConfig(&config.Config{})

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			require.NoError(tt, err)
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
