/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Config struct {
	messagePipe     bus.MessagePipeInterface
	configServices  map[string]service.ConfigServiceInterface
	instanceService service.InstanceServiceInterface
}

func NewConfig() *Config {
	return &Config{
		configServices:  make(map[string]service.ConfigServiceInterface), // key is instance id
		instanceService: service.NewInstanceService(),
	}
}

func (c *Config) Init(messagePipe bus.MessagePipeInterface) {
	c.messagePipe = messagePipe
}

func (c *Config) Close() {}

func (c *Config) Info() *bus.Info {
	return &bus.Info{
		Name: "config",
	}
}

func (c *Config) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC:
		if configurationStatus, ok := msg.Data.(*instances.ConfigurationStatus); !ok {
			slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
		} else {
			if configurationStatus.GetStatus() == instances.Status_SUCCESS {
				c.parseInstanceConfiguration(configurationStatus.CorrelationId, c.instanceService.GetInstance(configurationStatus.GetInstanceId()))
			}
		}

	case msg.Topic == bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC:
		if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
			slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
		} else {
			c.updateInstanceConfig(request)
		}
	}
}

func (c *Config) Subscriptions() []string {
	return []string{
		bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC,
		bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC,
	}
}

func (c *Config) parseInstanceConfiguration(correlationId string, instance *instances.Instance) {
	if c.configServices[instance.GetInstanceId()] == nil {
		c.configServices[instance.GetInstanceId()] = service.NewConfigService()
	}

	parsedConfig, err := c.configServices[instance.GetInstanceId()].ParseInstanceConfiguration(correlationId, instance)
	if err != nil {
		slog.Error("Unable to parse instance configuration", "correlationId", correlationId, "instanceId", instance.GetInstanceId(), "error", err)
	} else {
		switch config := parsedConfig.(type) {
		case model.NginxConfigContext:
			c.configServices[instance.GetInstanceId()].SetConfigContext(config)
		default:
			slog.Debug("Unknown config context", "configContext", config)
		}
		c.messagePipe.Process(&bus.Message{Topic: bus.INSTANCE_CONFIG_CONTEXT_TOPIC, Data: parsedConfig})
	}
}

func (c *Config) updateInstanceConfig(request *model.InstanceConfigUpdateRequest) {
	if c.configServices[request.Instance.GetInstanceId()] == nil {
		c.configServices[request.Instance.GetInstanceId()] = service.NewConfigService()
	}

	status := c.configServices[request.Instance.GetInstanceId()].UpdateInstanceConfiguration(request.CorrelationId, request.Location, request.Instance)
	c.messagePipe.Process(&bus.Message{Topic: bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC, Data: status})
}
