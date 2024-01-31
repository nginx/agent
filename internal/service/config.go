/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/service/config"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ConfigServiceInterface
type ConfigServiceInterface interface {
	SetConfigContext(instanceConfigContext any)
	UpdateInstanceConfiguration(correlationId, location string, instance *instances.Instance) error
	ParseInstanceConfiguration(correlationId string, instance *instances.Instance) (instanceConfigContext any, err error)
}

type ConfigService struct {
	configContext           any
	dataplaneConfigServices map[instances.Type]config.DataplaneConfig
}

func NewConfigService() *ConfigService {
	nginxConfigService := config.NewNginx()

	return &ConfigService{
		dataplaneConfigServices: map[instances.Type]config.DataplaneConfig{
			instances.Type_NGINX:                nginxConfigService,
			instances.Type_NGINX_PLUS:           nginxConfigService,
			instances.Type_NGINX_GATEWAY_FABRIC: config.NewNginxGatewayFabric(),
		},
	}
}

func (cs *ConfigService) SetConfigContext(instanceConfigContext any) {
	cs.configContext = instanceConfigContext
}

func (cs *ConfigService) UpdateInstanceConfiguration(_, _ string, _ *instances.Instance) error {
	return nil
}

func (cs *ConfigService) ParseInstanceConfiguration(_ string, instance *instances.Instance) (instanceConfigContext any, err error) {
	if conf, ok := cs.dataplaneConfigServices[instance.GetType()]; !ok {
		return nil, fmt.Errorf("unknown instance type %s", instance.Type)
	} else {
		return conf.ParseConfig(instance)
	}
}
