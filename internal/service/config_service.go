// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/client"

	"google.golang.org/protobuf/types/known/timestamppb"

	datasource "github.com/nginx/agent/v3/internal/datasource/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	service "github.com/nginx/agent/v3/internal/service/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ConfigServiceInterface
type ConfigServiceInterface interface {
	SetConfigContext(instanceConfigContext any)
	UpdateInstanceConfiguration(
		ctx context.Context,
		request *v1.ManagementPlaneRequest_ConfigApplyRequest,
	) (skippedFiles datasource.CacheContent, configStatus *instances.ConfigurationStatus)
	ParseInstanceConfiguration(
		ctx context.Context,
	) (instanceConfigContext any, err error)
	Rollback(ctx context.Context, skippedFiles datasource.CacheContent,
		request *v1.ManagementPlaneRequest_ConfigApplyRequest, instanceID string) error
}

type ConfigService struct {
	configContext any
	configService service.DataPlaneConfig
	instance      *v1.Instance
}

func NewConfigService(ctx context.Context, instance *v1.Instance, agentConfig *config.Config,
	configClient client.ConfigClient,
) *ConfigService {
	cs := &ConfigService{}

	switch instance.GetInstanceMeta().GetInstanceType() {
	case v1.InstanceMeta_INSTANCE_TYPE_NGINX, v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS:
		cs.configService = service.NewNginx(ctx, instance, agentConfig, configClient)
	case v1.InstanceMeta_INSTANCE_TYPE_UNSPECIFIED,
		v1.InstanceMeta_INSTANCE_TYPE_AGENT,
		v1.InstanceMeta_INSTANCE_TYPE_UNIT:
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

func (cs *ConfigService) Rollback(ctx context.Context, skippedFiles datasource.CacheContent,
	request *v1.ManagementPlaneRequest_ConfigApplyRequest,
	instanceID string,
) error {
	return cs.configService.Rollback(ctx, skippedFiles, request, instanceID)
}

func (cs *ConfigService) UpdateInstanceConfiguration(ctx context.Context,
	request *v1.ManagementPlaneRequest_ConfigApplyRequest,
) (skippedFiles datasource.CacheContent, configStatus *instances.ConfigurationStatus) {
	correlationID := logger.GetCorrelationID(ctx)

	skippedFiles, err := cs.configService.Write(ctx, request)
	if err != nil {
		slog.ErrorContext(ctx, "Error writing config", "error", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceMeta().GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Validate(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error validating config", "error", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceMeta().GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Apply(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error applying config and reloading nginx", "error", err)
		return skippedFiles, &instances.ConfigurationStatus{
			InstanceId:    cs.instance.GetInstanceMeta().GetInstanceId(),
			CorrelationId: correlationID,
			Status:        instances.Status_FAILED,
			Message:       err.Error(),
			Timestamp:     timestamppb.Now(),
		}
	}

	err = cs.configService.Complete(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Error updating instance file cache during config apply complete", "instance_id",
			cs.instance.GetInstanceMeta().GetInstanceId(), "error", err)
	}

	return skippedFiles, &instances.ConfigurationStatus{
		InstanceId:    cs.instance.GetInstanceMeta().GetInstanceId(),
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
		Timestamp:     timestamppb.Now(),
	}
}

func (cs *ConfigService) ParseInstanceConfiguration(
	ctx context.Context,
) (instanceConfigContext any, err error) {
	return cs.configService.ParseConfig(ctx)
}
