/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Config struct {
	messagePipe    bus.MessagePipeInterface
	configServices map[string]service.ConfigServiceInterface
}

func NewConfig() *Config {
	return &Config{
		configServices: make(map[string]service.ConfigServiceInterface), // key is instance id
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
	case msg.Topic == bus.INSTANCE_CONFIG_UPDATED_TOPIC:
		if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
			slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
		} else {
			c.parseInstanceConfiguration(request)
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
		bus.INSTANCE_CONFIG_UPDATED_TOPIC,
	}
}

func (c *Config) parseInstanceConfiguration(request *model.InstanceConfigUpdateRequest) {
	if c.configServices[request.Instance.GetInstanceId()] == nil {
		c.configServices[request.Instance.GetInstanceId()] = service.NewConfigService()
	}

	parsedConfig, err := c.configServices[request.Instance.GetInstanceId()].ParseInstanceConfiguration(request.CorrelationId, request.Instance)
	if err != nil {
		slog.Error("Unable to parse instance configuration", "correlationId", request.CorrelationId, "instanceId", request.Instance.InstanceId, "error", err)
	} else {
		switch config := parsedConfig.(type) {
		case model.NginxConfigContext:
			c.configServices[request.Instance.GetInstanceId()].SetConfigContext(config)
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

	err := c.configServices[request.Instance.GetInstanceId()].UpdateInstanceConfiguration(request.CorrelationId, request.Location, request.Instance)
	if err != nil {
		slog.Error("Unable to update instance configuration", "correlationId", request.CorrelationId, "instanceId", request.Instance.InstanceId, "error", err)
	} else {
		c.messagePipe.Process(&bus.Message{Topic: bus.INSTANCE_CONFIG_UPDATED_TOPIC, Data: request})
	}
}
