// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"testing"

	helpers "github.com/nginx/agent/v3/test"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service/servicefakes"
	"github.com/stretchr/testify/assert"
)

const (
	instanceID    = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
	correlationID = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
)

func TestConfig_Init(t *testing.T) {
	configPlugin := NewConfig(&config.Config{})
	err := configPlugin.Init(&bus.MessagePipe{})
	require.NoError(t, err)

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
		bus.InstanceConfigUpdateTopic,
		bus.InstancesTopic,
	}, subscriptions)
}

func TestConfig_Process(t *testing.T) {
	testInstance := &instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}

	nginxConfigContext := model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.log"}},
		ErrorLogs:  []*model.ErrorLog{{Name: "error.log"}},
	}

	instanceConfigUpdateRequest := &model.InstanceConfigUpdateRequest{
		Instance:      testInstance,
		Location:      "http://file-server.com",
		CorrelationID: correlationID,
	}

	configurationStatusProgress := helpers.CreateInProgressStatus()

	configurationStatus := helpers.CreateSuccessStatus()

	tests := []struct {
		name     string
		input    *bus.Message
		expected []*bus.Message
	}{
		{
			name: "Instance config updated",
			input: &bus.Message{
				Topic: bus.InstanceConfigUpdateTopic,
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
				Topic: bus.InstanceConfigUpdateTopic,
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
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  configurationStatusProgress,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  helpers.CreateSuccessStatus(),
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
		{
			name: "Instance topic request ",
			input: &bus.Message{
				Topic: bus.InstancesTopic,
				Data: []*instances.Instance{
					testInstance,
				},
			},
			expected: nil,
		},
	}

	for _, processTest := range tests {
		t.Run(processTest.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe(context.TODO())
			configPlugin := NewConfig(&config.Config{})

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			require.NoError(tt, err)
			messagePipe.Run()

			configService := &servicefakes.FakeConfigServiceInterface{}
			configService.ParseInstanceConfigurationReturns(nginxConfigContext, nil)
			configService.UpdateInstanceConfigurationReturns(nil, configurationStatus)

			instanceService := []*instances.Instance{testInstance}

			configPlugin.configServices[instanceID] = configService
			configPlugin.instances = instanceService

			configPlugin.Process(processTest.input)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, len(processTest.expected), len(messages))
		})
	}
}

func TestConfig_Update(t *testing.T) {
	agentConfig := config.Config{}
	instance := instances.Instance{
		InstanceId: instanceID,
		Type:       instances.Type_NGINX,
	}

	location := fmt.Sprintf("/instance/%s/files/", instanceID)
	request := model.InstanceConfigUpdateRequest{
		Instance:      &instance,
		Location:      location,
		CorrelationID: correlationID,
	}

	inProgressStatus := helpers.CreateInProgressStatus()
	successStatus := helpers.CreateSuccessStatus()
	failStatus := helpers.CreateFailStatus("error")
	rollbackInProgressStatus := helpers.CreateRollbackInProgressStatus()

	tests := []struct {
		name               string
		updateReturnStatus *instances.ConfigurationStatus
		rollbackReturns    error
		expected           []*bus.Message
	}{
		{
			name:               "success",
			updateReturnStatus: successStatus,
			rollbackReturns:    nil,
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  successStatus,
				},
			},
		},
		{
			name:               "fail and rollback",
			updateReturnStatus: failStatus,
			rollbackReturns:    nil,
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  failStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  rollbackInProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  helpers.CreateRollbackSuccessStatus(),
				},
			},
		},
		{
			name:               "rollback fails",
			updateReturnStatus: failStatus,
			rollbackReturns:    fmt.Errorf("rollback failed"),
			expected: []*bus.Message{
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  inProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  failStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  rollbackInProgressStatus,
				},
				{
					Topic: bus.InstanceConfigUpdateTopic,
					Data:  helpers.CreateRollbackFailStatus("rollback failed"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			messagePipe := bus.NewFakeMessagePipe(context.TODO())
			configPlugin := NewConfig(&agentConfig)

			err := messagePipe.Register(10, []bus.Plugin{configPlugin})
			require.NoError(tt, err)
			messagePipe.Run()

			configService := &servicefakes.FakeConfigServiceInterface{}
			configService.UpdateInstanceConfigurationReturns(make(map[string]*instances.File), test.updateReturnStatus)
			configService.RollbackReturns(test.rollbackReturns)

			instanceService := []*instances.Instance{&instance}
			configPlugin.configServices[instanceID] = configService
			configPlugin.instances = instanceService

			configPlugin.updateInstanceConfig(&request)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, len(test.expected), len(messages))
		})
	}
}
