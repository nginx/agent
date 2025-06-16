// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
	"testing"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/bus/busfakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nolint: dupl
func TestReadOnlyPlugin_Process_NginxConfigUpdateTopic(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/etc/nginx/nginx/conf", "")

	message := &model.NginxConfigContext{
		Files: []*mpi.File{
			{
				FileMeta: fileMeta,
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.UpdateOverviewReturns(&mpi.UpdateOverviewResponse{
		Overview: nil,
	}, nil)

	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	readPlugin := NewReadFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := readPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	readPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	readPlugin.Process(ctx, &bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateOverviewCallCount() == 1 },
		2*time.Second,
		10*time.Millisecond,
	)
}

func TestReadPlugin_ConnectionReset(t *testing.T) {
	ctx := context.Background()

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)

	newFakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	newGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	newGrpcConnection.FileServiceClientReturns(newFakeFileServiceClient)

	messagePipe := busfakes.NewFakeMessagePipe()

	readPlugin := NewReadFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := readPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	message := &bus.Message{Topic: bus.ConnectionResetTopic, Data: newGrpcConnection}

	readPlugin.Process(ctx, message)
	assert.Eventually(t,
		func() bool {
			return fakeGrpcConnection.CloseCallCount() == 1
		},
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Eventually(t,
		func() bool {
			return newGrpcConnection.FileServiceClientCallCount() == 1
		},
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, newGrpcConnection, readPlugin.conn)
}

// nolint: dupl
func TestReadPlugin_Process_ConfigUploadRequestTopic(t *testing.T) {
	ctx := context.Background()

	tempDir := os.TempDir()
	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	fileMeta := protos.FileMeta(testFile.Name(), "")

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{
						{
							FileMeta: fileMeta,
						},
						{
							FileMeta: fileMeta,
						},
					},
					ConfigVersion: &mpi.ConfigVersion{
						InstanceId: "123",
						Version:    "f33ref3d32d3c32d3a",
					},
				},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	readPlugin := NewReadFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := readPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	readPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	readPlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateFileCallCount() == 2 },
		2*time.Second,
		10*time.Millisecond,
	)

	messages := messagePipe.Messages()
	assert.Len(t, messages, 1)
	assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)

	dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_OK,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)
}

func TestReadPlugin_Process_ConfigUploadRequestTopic_Failure(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/unknown/file.conf", "")

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				Overview: &mpi.FileOverview{
					Files: []*mpi.File{
						{
							FileMeta: fileMeta,
						},
						{
							FileMeta: fileMeta,
						},
					},
					ConfigVersion: protos.CreateConfigVersion(),
				},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	readPlugin := NewReadFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := readPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	readPlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	readPlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.Messages()) == 1 },
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, 0, fakeFileServiceClient.UpdateFileCallCount())

	messages := messagePipe.Messages()
	assert.Len(t, messages, 1)

	assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)

	dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)
}

func TestReadPlugin_Subscriptions(t *testing.T) {
	readPlugin := NewReadFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(t, []string{
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
	}, readPlugin.Subscriptions())
}

func TestReadPlugin_Info(t *testing.T) {
	filePlugin := NewReadFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(t, "read", filePlugin.Info().Name)
}

func TestReadPlugin_Close(t *testing.T) {
	ctx := context.Background()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}

	filePlugin := NewReadFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	filePlugin.Close(ctx)

	assert.Equal(t, 1, fakeGrpcConnection.CloseCallCount())
}
