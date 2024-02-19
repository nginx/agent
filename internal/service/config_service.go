// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	service "github.com/nginx/agent/v3/internal/service/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ConfigServiceInterface
type ConfigServiceInterface interface {
	SetConfigContext(instanceConfigContext any)
	UpdateInstanceConfiguration(
		ctx context.Context,
		correlationID, location string,
		instance *instances.Instance,
	) *instances.ConfigurationStatus
	ParseInstanceConfiguration(
		correlationID string,
		instance *instances.Instance,
	) (instanceConfigContext any, err error)
}

type ConfigService struct {
	configContext any
	configService service.DataplaneConfig
}

func NewConfigService(instanceID string, agentConfig *config.Config, instanceType instances.Type) *ConfigService {
	cs := &ConfigService{}

	switch instanceType {
	case instances.Type_NGINX:
		cs.configService = service.NewNginx(instanceID, agentConfig)
	case instances.Type_NGINX_GATEWAY_FABRIC:
		cs.configService = service.NewNginxGatewayFabric()
	case instances.Type_NGINX_PLUS:
		slog.Warn("Not * Implemented")
	case instances.Type_UNKNOWN:
		slog.Warn("Not Implemented")
	}

	return cs
}

func (cs *ConfigService) SetConfigContext(instanceConfigContext any) {
	cs.configContext = instanceConfigContext
}

func (cs *ConfigService) UpdateInstanceConfiguration(ctx context.Context, correlationID, location string,
	instance *instances.Instance,
) *instances.ConfigurationStatus {
	// remove when tenantID is being set
	tenantID := "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

	_, err := cs.configService.Write(ctx, location, tenantID, instance.GetInstanceId())
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       fmt.Sprintf("%s", err),
		}
	}

	err = cs.configService.Validate(instance)

	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       fmt.Sprintf("%s", err),
		}
	}

	err = cs.configService.Apply(instance)
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       fmt.Sprintf("%s", err),
		}
	}

	err = cs.configService.Complete()
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       fmt.Sprintf("%s", err),
		}
	}

	return &instances.ConfigurationStatus{
		InstanceId:    instance.GetInstanceId(),
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
	}
}

func (cs *ConfigService) ParseInstanceConfiguration(
	_ string,
	instance *instances.Instance,
) (instanceConfigContext any, err error) {
	return cs.configService.ParseConfig(instance)
}
