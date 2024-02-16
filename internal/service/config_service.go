// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"fmt"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	service "github.com/nginx/agent/v3/internal/service/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ConfigServiceInterface
type ConfigServiceInterface interface {
	SetConfigContext(instanceConfigContext any)
	UpdateInstanceConfiguration(
		correlationID, location string,
		instance *instances.Instance,
	) *instances.ConfigurationStatus
	ParseInstanceConfiguration(
		correlationID string,
		instance *instances.Instance,
	) (instanceConfigContext any, err error)
}

type ConfigService struct {
	configContext           any
	dataplaneConfigServices map[instances.Type]service.DataplaneConfig
}

func NewConfigService(instanceID string, agentConfig *config.Config) *ConfigService {
	nginxConfigService := service.NewNginx(instanceID, agentConfig)

	return &ConfigService{
		dataplaneConfigServices: map[instances.Type]service.DataplaneConfig{
			instances.Type_NGINX:                nginxConfigService,
			instances.Type_NGINX_PLUS:           nginxConfigService,
			instances.Type_NGINX_GATEWAY_FABRIC: service.NewNginxGatewayFabric(),
		},
	}
}

func (cs *ConfigService) SetConfigContext(instanceConfigContext any) {
	cs.configContext = instanceConfigContext
}

func (*ConfigService) UpdateInstanceConfiguration(_, _ string, _ *instances.Instance) *instances.ConfigurationStatus {
	return nil
}

func (cs *ConfigService) ParseInstanceConfiguration(
	_ string,
	instance *instances.Instance,
) (instanceConfigContext any, err error) {
	conf, ok := cs.dataplaneConfigServices[instance.GetType()]

	if !ok {
		return nil, fmt.Errorf("unknown instance type %s", instance.GetType())
	}

	return conf.ParseConfig(instance)
}
