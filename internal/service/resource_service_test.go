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
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

func TestResourceService_GetResource(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		isContainer      bool
		expectedResource *v1.Resource
	}{
		{
			isContainer:      true,
			expectedResource: protos.GetContainerizedResource(),
		},
		{
			isContainer:      false,
			expectedResource: protos.GetHostResource(),
		},
	}
	for _, tc := range testCases {
		mockInfo := &hostfakes.FakeInfoInterface{}
		if tc.isContainer {
			mockInfo.ContainerInfoReturns(
				&v1.Resource_ContainerInfo{
					ContainerInfo: tc.expectedResource.GetContainerInfo(),
				},
			)
		} else {
			mockInfo.HostInfoReturns(
				&v1.Resource_HostInfo{
					HostInfo: tc.expectedResource.GetHostInfo(),
				},
			)
		}

		mockInfo.IsContainerReturns(tc.isContainer)

		resourceService := NewResourceService()
		resourceService.info = mockInfo

		resource := resourceService.GetResource(ctx)
		assert.Equal(t, tc.expectedResource.GetResourceId(), resource.GetResourceId())
		assert.Empty(t, resource.GetInstances())

		if tc.isContainer {
			assert.Equal(t, tc.expectedResource.GetContainerInfo(), resource.GetContainerInfo())
		} else {
			assert.Equal(t, tc.expectedResource.GetHostInfo(), resource.GetHostInfo())
		}
	}
}
