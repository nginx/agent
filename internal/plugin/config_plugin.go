// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

// remove when tenantID is being set
const tenantID = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

type Config struct {
	messagePipe    bus.MessagePipeInterface
	configServices map[string]service.ConfigServiceInterface
	instances      []*instances.Instance
	agentConfig    *config.Config
}

func NewConfig(agentConfig *config.Config) *Config {
	return &Config{
		configServices: make(map[string]service.ConfigServiceInterface), // key is instance id
		instances:      []*instances.Instance{},
		agentConfig:    agentConfig,
	}
}

func (c *Config) Init(messagePipe bus.MessagePipeInterface) error {
	c.messagePipe = messagePipe
	return nil
}

func (*Config) Close() error { return nil }

func (*Config) Info() *bus.Info {
	return &bus.Info{
		Name: "config",
	}
}

func (c *Config) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstanceConfigUpdateTopic:
		c.processConfigurationStatus(msg)
	case msg.Topic == bus.InstanceConfigUpdateRequestTopic:
		c.processInstanceConfigUpdateRequest(msg)
	case msg.Topic == bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*instances.Instance); ok {
			c.instances = newInstances
		}
	}
}

func (*Config) Subscriptions() []string {
	return []string{
		bus.InstanceConfigUpdateRequestTopic,
		bus.InstanceConfigUpdateTopic,
		bus.InstancesTopic,
	}
}

func (c *Config) processConfigurationStatus(msg *bus.Message) {
	if configurationStatus, ok := msg.Data.(*instances.ConfigurationStatus); !ok {
		slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
	} else if configurationStatus.GetStatus() == instances.Status_SUCCESS {
		c.parseInstanceConfiguration(
			configurationStatus.GetCorrelationId(),
			c.GetInstance(configurationStatus.GetInstanceId()),
		)
	}
}

func (c *Config) GetInstance(instanceID string) *instances.Instance {
	for _, instanceEntity := range c.instances {
		if instanceEntity.GetInstanceId() == instanceID {
			return instanceEntity
		}
	}

	return nil
}

func (c *Config) processInstanceConfigUpdateRequest(msg *bus.Message) {
	if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
		slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
	} else {
		c.updateInstanceConfig(request)
	}
}

func (c *Config) parseInstanceConfiguration(correlationID string, instance *instances.Instance) {
	if c.configServices[instance.GetInstanceId()] == nil {
		c.configServices[instance.GetInstanceId()] = service.NewConfigService(instance,
			c.agentConfig)
	}

	parsedConfig, err := c.configServices[instance.GetInstanceId()].ParseInstanceConfiguration(correlationID)
	if err != nil {
		slog.Error(
			"Unable to parse instance configuration",
			"correlation_id", correlationID,
			"instance_id", instance.GetInstanceId(),
			"error", err,
		)
	} else {
		switch configContext := parsedConfig.(type) {
		case model.NginxConfigContext:
			c.configServices[instance.GetInstanceId()].SetConfigContext(configContext)
		default:
			slog.Debug("Unknown config context", "config_context", configContext)
		}
		c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigContextTopic, Data: parsedConfig})
	}
}

func (c *Config) updateInstanceConfig(request *model.InstanceConfigUpdateRequest) {
	slog.Debug("Updating instance configuration")
	instanceID := request.Instance.GetInstanceId()
	if c.configServices[instanceID] == nil {
		c.configServices[instanceID] = service.NewConfigService(request.Instance, c.agentConfig)
	}

	inProgressStatus := &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: request.CorrelationID,
		Status:        instances.Status_IN_PROGRESS,
		Timestamp:     timestamppb.Now(),
		Message:       "Instance configuration update in progress",
	}
	c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: inProgressStatus})

	skippedFiles, status := c.configServices[request.Instance.GetInstanceId()].UpdateInstanceConfiguration(
		c.messagePipe.Context(),
		request.CorrelationID,
		request.Location,
	)
	c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: status})

	if status.GetStatus() == instances.Status_FAILED {
		rollbackInProgress := &instances.ConfigurationStatus{
			InstanceId:    instanceID,
			CorrelationId: request.CorrelationID,
			Status:        instances.Status_ROLLBACK_IN_PROGRESS,
			Timestamp:     timestamppb.Now(),
			Message:       "Rollback in progress",
		}
		c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackInProgress})

		err := c.configServices[instanceID].Rollback(c.messagePipe.Context(), skippedFiles,
			request.Location, tenantID, instanceID)
		if err != nil {
			rollbackFailed := &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: request.CorrelationID,
				Status:        instances.Status_ROLLBACK_FAILED,
				Timestamp:     timestamppb.Now(),
				Message:       err.Error(),
			}
			c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackFailed})
		} else {
			rollbackComplete := &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: request.CorrelationID,
				Status:        instances.Status_ROLLBACK_SUCCESS,
				Timestamp:     timestamppb.Now(),
				Message:       "Rollback successful",
			}
			c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackComplete})
		}
	}
}
