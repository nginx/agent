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
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePlugin_Info(t *testing.T) {
	filePlugin := NewFilePlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(t, "file", filePlugin.Info().Name)
}

func TestFilePlugin_Close(t *testing.T) {
	ctx := context.Background()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}

	filePlugin := NewFilePlugin(types.GetAgentConfig(), fakeGrpcConnection)
	filePlugin.Close(ctx)

	assert.Equal(t, 1, fakeGrpcConnection.CloseCallCount())
}

func TestFilePlugin_Subscriptions(t *testing.T) {
	filePlugin := NewFilePlugin(types.GetAgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(
		t,
		[]string{
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
		},
		filePlugin.Subscriptions(),
	)
}

func TestFilePlugin_Process_NginxConfigUpdateTopic(t *testing.T) {
	ctx := context.Background()

	fileMeta, fileMetaError := protos.GetFileMeta("/etc/nginx/nginx/conf")
	require.NoError(t, fileMetaError)

	message := &model.NginxConfigContext{
		Files: []*mpi.File{
			{
				FileMeta: fileMeta,
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := bus.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.GetAgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateOverviewCallCount() == 1 },
		2*time.Second,
		10*time.Millisecond,
	)
}

func TestFilePlugin_Process_ConfigUploadRequestTopic(t *testing.T) {
	ctx := context.Background()

	tempDir := os.TempDir()
	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	fileMeta, fileMetaError := protos.GetFileMeta(testFile.Name())
	require.NoError(t, fileMetaError)

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				InstanceId: "123",
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
	messagePipe := bus.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.GetAgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateFileCallCount() == 2 },
		2*time.Second,
		10*time.Millisecond,
	)

	messages := messagePipe.GetMessages()
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

func TestFilePlugin_Process_ConfigUploadRequestTopic_Failure(t *testing.T) {
	ctx := context.Background()

	fileMeta, fileMetaError := protos.GetFileMeta("/unknown/file.conf")
	require.NoError(t, fileMetaError)

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				InstanceId: "123",
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
	messagePipe := bus.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.GetAgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.GetMessages()) == 2 },
		2*time.Second,
		10*time.Millisecond,
	)

	assert.Equal(t, 0, fakeFileServiceClient.UpdateFileCallCount())

	messages := messagePipe.GetMessages()
	assert.Len(t, messages, 2)
	assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)

	dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_ERROR,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)

	assert.Equal(t, bus.DataPlaneResponseTopic, messages[1].Topic)

	dataPlaneResponse, ok = messages[1].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)
	assert.Equal(
		t,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		dataPlaneResponse.GetCommandResponse().GetStatus(),
	)
}
