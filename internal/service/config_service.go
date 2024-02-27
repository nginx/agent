// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
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
	) *instances.ConfigurationStatus
	ParseInstanceConfiguration(
		correlationID string,
	) (instanceConfigContext any, err error)
}

type ConfigService struct {
	configContext any
	configService service.DataPlaneConfig
	instance      *instances.Instance
}

func NewConfigService(instance *instances.Instance, agentConfig *config.Config) *ConfigService {
	cs := &ConfigService{}

	switch instance.GetType() {
	case instances.Type_NGINX:
		cs.configService = service.NewNginx(instance, agentConfig)
	case instances.Type_NGINX_GATEWAY_FABRIC:
		cs.configService = service.NewNginxGatewayFabric()
	case instances.Type_NGINX_PLUS:
		fallthrough
	case instances.Type_UNKNOWN:
		fallthrough
	default:
		slog.Warn("Not Implemented")
	}

	cs.instance = instance

	return cs
}

func (cs *ConfigService) SetConfigContext(instanceConfigContext any) {
	cs.configContext = instanceConfigContext
}

func (cs *ConfigService) UpdateInstanceConfiguration(ctx context.Context, correlationID, location string,
) *instances.ConfigurationStatus {
	// remove when tenantID is being set
	tenantID := "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

	_, err := cs.configService.Write(ctx, location, tenantID)
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
		}
	}

	err = cs.configService.Validate()
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
		}
	}

	err = cs.configService.Apply()
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
		}
	}

	err = cs.configService.Complete()
	if err != nil {
		// Rollback
		return &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
		}
	}

	return &instances.ConfigurationStatus{
		InstanceId:    cs.instance.GetInstanceId(),
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
	}
}

func (cs *ConfigService) ParseInstanceConfiguration(
	_ string,
) (instanceConfigContext any, err error) {
	return cs.configService.ParseConfig()
}
