/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/config"
	configWriter "github.com/nginx/agent/v3/internal/datasource/config"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/datasource/os"

	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/types/known/timestamppb"
)

var testInstances = []*instances.Instance{
	{
		InstanceId: "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		Type:       instances.Type_NGINX,
	},
}

func TestInstanceService_UpdateInstances(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.UpdateInstances(testInstances)
	assert.Equal(t, testInstances, instanceService.instances)
}

func TestInstanceService_GetInstances(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.UpdateInstances(testInstances)
	assert.Equal(t, testInstances, instanceService.GetInstances())
}

func TestUpdateInstanceConfiguration(t *testing.T) {
	_, instanceId, err := createTestIds()
	assert.NoError(t, err)

	location := "/tmp/test.conf"
	cachePath := fmt.Sprintf("/tmp/%s/cache.json", instanceId.String())

	client := config.Client{
		Timeout: time.Second,
	}
	fakeConfig := config.FakeConfigInterface{}
	fakeConfig.GetClientReturns(client)

	files := os.FakeFilesInterface{}
	
	cacheTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	assert.NoError(t, err)

	cacheTime2, err := createProtoTime("2024-01-08T12:22:21Z")
	assert.NoError(t, err)

	previouseFileCache := os.FileCache{
		"/tmp/nginx/nginx.conf": {
			LastModified: cacheTime1,
			Path:         "/tmp/nginx/nginx.conf",
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
		"/tmp/test.conf": {
			LastModified: cacheTime2,
			Path:         "/tmp/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	updateTimeFile1, err := createProtoTime("2024-01-08T14:22:23Z")
	
	currentFileCache := os.FileCache{
		"/tmp/nginx/nginx.conf": {
			LastModified: cacheTime1,
			Path:         "/tmp/nginx/nginx.conf",
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
		"/tmp/test.conf": {
			LastModified: updateTimeFile1,
			Path:         "/tmp/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	skippedFiles := make(map[string]struct{})
	skippedFiles["/tmp/nginx/nginx.conf"] = struct{}{}

	files.ReadInstanceCacheReturns(previouseFileCache, nil)

	configWriter := configWriter.FakeConfigWriterInterface{}


	configWriter.WriteReturns(currentFileCache, skippedFiles, nil)
	
	nginxConfig := nginx.FakeNginxConfigInterface{}

	nginxConfig.ValidateReturns(nil)

	nginxConfig.ReloadReturns(nil)

	files.UpdateCacheReturns(nil)

	instanceService := NewInstanceService()

	instanceService.UpdateInstances(testInstances)

	correlationId, err := instanceService.UpdateInstanceConfiguration(instanceId.String(), location, cachePath)

	assert.NoError(t, err)
	assert.NotEmpty(t, correlationId)

}

func createTestIds() (uuid.UUID, uuid.UUID, error) {
	tenantId, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	if err != nil {
		fmt.Printf("Error creating tenantId: %v", err)
	}

	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	if err != nil {
		fmt.Printf("Error creating instanceId: %v", err)
	}

	return tenantId, instanceId, err
}

func createProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	time, err := time.Parse(time.RFC3339, timeString)
	protoTime := timestamppb.New(time)

	return protoTime, err
}