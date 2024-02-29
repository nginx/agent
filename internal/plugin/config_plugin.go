// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

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
	case msg.Topic == bus.InstanceConfigUpdateCompleteTopic:
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
		bus.InstanceConfigUpdateCompleteTopic,
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

	status := c.configServices[request.Instance.GetInstanceId()].UpdateInstanceConfiguration(
		c.messagePipe.Context(),
		request.CorrelationID,
		request.Location,
	)
	c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateCompleteTopic, Data: status})
}
