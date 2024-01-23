/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/nginx/agent/v3/internal/config"
	configWriter "github.com/nginx/agent/v3/internal/datasource/config"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/datasource/os"
)

const (
	tenantId = ("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_instance.go . InstanceServiceInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/service mock_instance.go | sed -e s\\/service\\\\.\\/\\/g > mock_instance_fixed.go"
//go:generate mv mock_instance_fixed.go mock_instance.go
type InstanceServiceInterface interface {
	UpdateInstances(newInstances []*instances.Instance)
	GetInstances() []*instances.Instance
	UpdateInstanceConfiguration(instanceId string, location string, cachePath string) (string, error)
}

type InstanceServiceParameters struct {
	files        os.FilesInterface
	config       config.ConfigInterface
	nginxConfig  nginx.NginxConfig
	configWriter configWriter.ConfigWriter
}

type InstanceService struct {
	instances          []*instances.Instance
	nginxInstances     map[string]*instances.Instance
	configWriterParams configWriter.ConfigWriterParameters
}

// instanceServiceParameters *InstanceServiceParameters
func NewInstanceService() *InstanceService {
	// if instanceServiceParameters.files == nil {
	// 	instanceServiceParameters.files = os.
	// }

	return &InstanceService{
		nginxInstances: make(map[string]*instances.Instance),
		configWriterParams: configWriter.ConfigWriterParameters{
			Client: configWriter.Client{
				Timeout: config.GetConfig().Client.Timeout,
			},
		},
	}
}

func (is *InstanceService) UpdateInstances(newInstances []*instances.Instance) {
	is.instances = newInstances
	if is.instances != nil {
		for _, instance := range is.instances {
			if instance.Type == instances.Type_NGINX || instance.Type == instances.Type_NGINXPLUS {
				is.nginxInstances[instance.InstanceId] = instance
			}
		}
	}
}

func (is *InstanceService) GetInstances() []*instances.Instance {
	return is.instances
}

func (is *InstanceService) UpdateInstanceConfiguration(instanceId string, location string, cachePath string) (correlationId string, err error) {
	// TODO: Remove when getting tenantId
	exampleTenantId, err := uuid.Parse(tenantId)
	if err != nil {
		fmt.Printf("Error creating tenantId: %v", err)
	}

	correlationId = uuid.New().String()
	if _, ok := is.nginxInstances[instanceId]; ok {
		if cachePath == "" {
			cachePath = fmt.Sprintf("/var/lib/nginx-agent/config/%v/cache.json", instanceId)
		}

		previousCache, err := os.ReadInstanceCache(cachePath)
		if err != nil {
			slog.Info("Error ", "err", err, "cache", cachePath, "previousCache", previousCache)
			return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Failed to Read cache for instance with id %s", instanceId)}
		}

		configWriter := configWriter.NewConfigWriter(&is.configWriterParams)

		// Skipped files currently not being used will be changed when doing rollback
		nginxConfig := nginx.NewNginxConfig(instanceId, *configWriter)

		currentCache, _, err := nginxConfig.Write(previousCache, location, exampleTenantId)
		if err != nil {
			return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Failed to update config for instance with id %s", instanceId)}
		}

		err = nginxConfig.Validate()
		if err != nil {
			return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Config validation failed for instance with id %s", instanceId)}
		}

		err = nginxConfig.Reload()
		if err != nil {
			return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Failed to reload NGINX for instance with id %s", instanceId)}
		}

		err = os.UpdateCache(currentCache, cachePath)
		if err != nil {
			return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("Failed to update cache for instance with id %s", instanceId)}
		}

	} else {
		return correlationId, &common.RequestError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("unable to find instance with id %s", instanceId)}
	}
	return correlationId, err
}
