// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FakeSubscribeClient struct {
	grpc.ClientStream
}

func (*FakeSubscribeClient) Send(*mpi.DataPlaneResponse) error {
	return nil
}

// nolint: nilnil
func (*FakeSubscribeClient) Recv() (*mpi.ManagementPlaneRequest, error) {
	return nil, nil
}

func TestCommandService_UpdateDataPlaneStatus(t *testing.T) {
	ctx := context.Background()

	fakeSubscribeClient := &FakeSubscribeClient{}

	commandServiceClient := &v1fakes.FakeCommandServiceClient{}
	commandServiceClient.SubscribeReturns(fakeSubscribeClient, nil)

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.GetAgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)
	defer commandService.CancelSubscription(ctx)

	err := commandService.UpdateDataPlaneStatus(ctx, protos.GetHostResource())

	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.CreateConnectionCallCount())
	assert.Equal(t, 1, commandServiceClient.UpdateDataPlaneStatusCallCount())
}

func TestCommandService_UpdateDataPlaneHealth(t *testing.T) {
	ctx := context.Background()
	commandServiceClient := &v1fakes.FakeCommandServiceClient{}

	commandService := NewCommandService(
		ctx,
		commandServiceClient,
		types.GetAgentConfig(),
		make(chan *mpi.ManagementPlaneRequest),
	)

	// connection not created yet
	err := commandService.UpdateDataPlaneHealth(ctx, protos.GetInstanceHealths())

	require.Error(t, err)
	assert.Equal(t, 0, commandServiceClient.UpdateDataPlaneHealthCallCount())

	// connection created
	err = commandService.createConnection(ctx, protos.GetHostResource())
	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.CreateConnectionCallCount())

	err = commandService.UpdateDataPlaneHealth(ctx, protos.GetInstanceHealths())

	require.NoError(t, err)
	assert.Equal(t, 1, commandServiceClient.UpdateDataPlaneHealthCallCount())
}
