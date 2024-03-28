// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

// remove when tenantID is being set
const tenantID = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

type Config struct {
	messagePipe    bus.MessagePipeInterface
	configServices map[string]service.ConfigServiceInterface
	instances      []*v1.Instance
	agentConfig    *config.Config
}

func NewConfig(agentConfig *config.Config) *Config {
	return &Config{
		configServices: make(map[string]service.ConfigServiceInterface), // key is instance id
		instances:      []*v1.Instance{},
		agentConfig:    agentConfig,
	}
}

func (c *Config) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting config plugin")
	c.messagePipe = messagePipe

	return nil
}

func (*Config) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing config plugin")

	return nil
}

func (*Config) Info() *bus.Info {
	return &bus.Info{
		Name: "config",
	}
}

func (c *Config) Process(ctx context.Context, msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstanceConfigUpdateTopic:
		c.processConfigurationStatus(ctx, msg)
	case msg.Topic == bus.InstanceConfigUpdateRequestTopic:
		c.processInstanceConfigUpdateRequest(ctx, msg)
	case msg.Topic == bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
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

func (c *Config) processConfigurationStatus(ctx context.Context, msg *bus.Message) {
	if configurationStatus, ok := msg.Data.(*instances.ConfigurationStatus); !ok {
		slog.DebugContext(ctx, "Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
	} else if configurationStatus.GetStatus() == instances.Status_SUCCESS {
		c.parseInstanceConfiguration(
			ctx,
			c.GetInstance(configurationStatus.GetInstanceId()),
		)
	}
}

func (c *Config) GetInstance(instanceID string) *v1.Instance {
	for _, instanceEntity := range c.instances {
		if instanceEntity.GetInstanceMeta().GetInstanceId() == instanceID {
			return instanceEntity
		}
	}

	return nil
}

func (c *Config) processInstanceConfigUpdateRequest(ctx context.Context, msg *bus.Message) {
	if request, ok := msg.Data.(*model.InstanceConfigUpdateRequest); !ok {
		slog.DebugContext(ctx, "Unknown message processed by config service", "topic", msg.Topic, "message", msg.Data)
	} else {
		c.updateInstanceConfig(ctx, request)
	}
}

func (c *Config) parseInstanceConfiguration(ctx context.Context, instance *v1.Instance) {
	instanceID := instance.GetInstanceMeta().GetInstanceId()
	if c.configServices[instanceID] == nil {
		c.configServices[instanceID] = service.NewConfigService(instance, c.agentConfig)
	}

	parsedConfig, err := c.configServices[instanceID].ParseInstanceConfiguration(ctx)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Unable to parse instance configuration",
			"instance_id", instanceID,
			"error", err,
		)
	} else {
		switch configContext := parsedConfig.(type) {
		case model.NginxConfigContext:
			c.configServices[instanceID].SetConfigContext(configContext)
		default:
			slog.DebugContext(ctx, "Unknown config context", "config_context", configContext)
		}
		c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigContextTopic, Data: parsedConfig})
	}
}

func (c *Config) updateInstanceConfig(ctx context.Context, request *model.InstanceConfigUpdateRequest) {
	slog.DebugContext(ctx, "Updating instance configuration")

	correlationID := logger.GetCorrelationID(ctx)

	instanceID := request.Instance.GetInstanceMeta().GetInstanceId()
	if c.configServices[instanceID] == nil {
		c.configServices[instanceID] = service.NewConfigService(request.Instance, c.agentConfig)
	}

	inProgressStatus := &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_IN_PROGRESS,
		Timestamp:     timestamppb.Now(),
		Message:       "Instance configuration update in progress",
	}
	c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: inProgressStatus})

	skippedFiles, status := c.configServices[instanceID].UpdateInstanceConfiguration(
		ctx,
		request.Location,
	)
	c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: status})

	if status.GetStatus() == instances.Status_FAILED {
		rollbackInProgress := &instances.ConfigurationStatus{
			InstanceId:    instanceID,
			CorrelationId: correlationID,
			Status:        instances.Status_ROLLBACK_IN_PROGRESS,
			Timestamp:     timestamppb.Now(),
			Message:       "Rollback in progress",
		}
		c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackInProgress})

		err := c.configServices[instanceID].Rollback(ctx, skippedFiles, request.Location, tenantID, instanceID)
		if err != nil {
			rollbackFailed := &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_ROLLBACK_FAILED,
				Timestamp:     timestamppb.Now(),
				Message:       err.Error(),
			}
			c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackFailed})
		} else {
			rollbackComplete := &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_ROLLBACK_SUCCESS,
				Timestamp:     timestamppb.Now(),
				Message:       "Rollback successful",
			}
			c.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstanceConfigUpdateTopic, Data: rollbackComplete})
		}
	}
}
