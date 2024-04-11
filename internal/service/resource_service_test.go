// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package service

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/hostfakes"
	"github.com/stretchr/testify/assert"
)

func TestResourceService_GetResource(t *testing.T) {
	ctx := context.Background()

	containerInfo := &v1.Resource_ContainerInfo{
		ContainerInfo: &v1.ContainerInfo{
			Id: "123",
		},
	}

	hostInfo := &v1.Resource_HostInfo{
		HostInfo: &v1.HostInfo{
			Id:       "123",
			Hostname: "example.com",
			ReleaseInfo: &v1.ReleaseInfo{
				Codename:  "linux",
				Id:        "ubuntu",
				Name:      "debian",
				VersionId: "2.34.2",
				Version:   "1.3.34",
			},
		},
	}

	mockInfo := &hostfakes.FakeInfoInterface{}
	mockInfo.GetContainerInfoReturns(containerInfo)
	mockInfo.GetHostInfoReturns(hostInfo)

	resourceService := NewResourceService()
	resourceService.info = mockInfo

	// Test Container
	mockInfo.IsContainerReturns(true)

	resource := resourceService.GetResource(ctx)

	assert.Equal(t, &v1.Resource{Id: "123", Info: containerInfo}, resource)

	// Test VM
	mockInfo.IsContainerReturns(false)

	resource = resourceService.GetResource(ctx)

	assert.Equal(t, &v1.Resource{Id: "123", Info: hostInfo}, resource)
}
