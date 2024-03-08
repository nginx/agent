// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/timestamppb"

	datasource "github.com/nginx/agent/v3/internal/datasource/config"

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
	) (skippedFiles datasource.CacheContent, configStatus *instances.ConfigurationStatus)
	ParseInstanceConfiguration(
		correlationID string,
	) (instanceConfigContext any, err error)
	Rollback(ctx context.Context, skippedFiles datasource.CacheContent, filesURL, tenantID, instanceID string) error
}

type ConfigService struct {
	configContext any
	configService service.DataPlaneConfig
	instance      *instances.Instance
}

func NewConfigService(instance *instances.Instance, agentConfig *config.Config) *ConfigService {
	cs := &ConfigService{}

	switch instance.GetType() {
	case instances.Type_NGINX, instances.Type_NGINX_PLUS:
		cs.configService = service.NewNginx(instance, agentConfig)
	case instances.Type_NGINX_GATEWAY_FABRIC:
		cs.configService = service.NewNginxGatewayFabric()
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

func (cs *ConfigService) Rollback(ctx context.Context, skippedFiles datasource.CacheContent, filesURL,
	tenantID, instanceID string,
) error {
	err := cs.configService.Rollback(ctx, skippedFiles, filesURL, tenantID, instanceID)
	return err
}

func (cs *ConfigService) UpdateInstanceConfiguration(ctx context.Context, correlationID, location string,
) (skippedFiles datasource.CacheContent, configStatus *instances.ConfigurationStatus) {
	// remove when tenantID is being set
	tenantID := "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

	skippedFiles, err := cs.configService.Write(ctx, location, tenantID)
	if err != nil {
		slog.Error("Error writing config", "err", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Validate()
	if err != nil {
		slog.Error("Error validating config", "err", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Apply()
	if err != nil {
		slog.Error("Error applying config and reloading nginx", "err", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Complete()
	if err != nil {
		slog.Error("error updating instance file cache during config apply complete", "instance_id",
			cs.instance.GetInstanceId(), "err", err)
	}

	return skippedFiles, &instances.ConfigurationStatus{
		InstanceId:    cs.instance.GetInstanceId(),
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
		Timestamp:     timestamppb.Now(),
	}
}

func (cs *ConfigService) ParseInstanceConfiguration(
	_ string,
) (instanceConfigContext any, err error) {
	return cs.configService.ParseConfig()
}
