// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

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

func (*Config) Close() {}

func (*Config) Info() *bus.Info {
	return &bus.Info{
		Name: "config",
	}
}

func (c *Config) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstanceConfigUpdatedTopic:
		if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
			slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
		} else {
			c.parseInstanceConfiguration(request)
		}

	case msg.Topic == bus.InstanceConfigUpdateRequestTopic:
		if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
			slog.Debug("Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
		} else {
			c.updateInstanceConfig(request)
		}
	}
}

func (*Config) Subscriptions() []string {
	return []string{
		bus.InstanceConfigUpdateRequestTopic,
		bus.InstanceConfigUpdatedTopic,
	}
}

func (c *Config) parseInstanceConfiguration(request *model.InstanceConfigUpdateRequest) {
	instanceID := request.Instance.GetInstanceId()
	if c.configServices[instanceID] == nil {
		c.configServices[instanceID] = service.NewConfigService()
	}

	configService := c.configServices[instanceID]

	parsedConfig, err := configService.ParseInstanceConfiguration(request.CorrelationID, request.Instance)
	if err != nil {
		slog.Error(
			"Unable to parse instance configuration",
			"correlationID", request.CorrelationID,
			"instanceID", instanceID,
			"error", err,
		)
	} else {
		switch config := parsedConfig.(type) {
		case model.NginxConfigContext:
			configService.SetConfigContext(config)
		default:
			slog.Debug("Unknown config context", "configContext", config)
		}
		c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigContextTopic, Data: parsedConfig})
	}
}

func (c *Config) updateInstanceConfig(request *model.InstanceConfigUpdateRequest) {
	instanceID := request.Instance.GetInstanceId()
	if c.configServices[instanceID] == nil {
		c.configServices[instanceID] = service.NewConfigService()
	}

	configService := c.configServices[instanceID]

	err := configService.UpdateInstanceConfiguration(request.CorrelationID, request.Location, request.Instance)
	if err != nil {
		slog.Error(
			"Unable to update instance configuration",
			"correlationId", request.CorrelationID,
			"instanceId", instanceID,
			"error", err,
		)
	} else {
		c.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdatedTopic, Data: request})
	}
}
