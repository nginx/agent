// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/file/filefakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePlugin_Info(t *testing.T) {
	filePlugin := NewFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(t, "file", filePlugin.Info().Name)
}

func TestFilePlugin_Close(t *testing.T) {
	ctx := context.Background()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	filePlugin.Close(ctx)

	assert.Equal(t, 1, fakeGrpcConnection.CloseCallCount())
}

func TestFilePlugin_Subscriptions(t *testing.T) {
	filePlugin := NewFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	assert.Equal(
		t,
		[]string{
			bus.ConnectionCreatedTopic,
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
			bus.ConfigApplyRequestTopic,
			bus.ConfigApplyFailedTopic,
			bus.ConfigApplySuccessfulTopic,
			bus.RollbackCompleteTopic,
		},
		filePlugin.Subscriptions(),
	)
}

func TestFilePlugin_Process_NginxConfigUpdateTopic(t *testing.T) {
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
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := bus.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	filePlugin.Process(ctx, &bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: message})

	assert.Eventually(
		t,
		func() bool { return fakeFileServiceClient.UpdateOverviewCallCount() == 1 },
		2*time.Second,
		10*time.Millisecond,
	)
}

func TestFilePlugin_Process_ConfigApplyRequestTopic(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	addAction := mpi.File_FILE_ACTION_ADD

	filePath := fmt.Sprintf("%s/nginx.conf", tempDir)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: protos.CreateConfigApplyRequest(protos.FileOverview(filePath, fileHash, &addAction)),
		},
	}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	tests := []struct {
		configApplyReturnsErr error
		message               *mpi.ManagementPlaneRequest
		name                  string
	}{
		{
			name:                  "Test 1 - Success",
			configApplyReturnsErr: nil,
			message:               message,
		},
		{
			name:                  "Test 2 - Fail, Rollback",
			configApplyReturnsErr: &RollbackRequiredError{Err: fmt.Errorf("something went wrong")},
			message:               message,
		},
		{
			name:                  "Test 3 - Fail, No Rollback",
			configApplyReturnsErr: fmt.Errorf("something went wrong"),
			message:               message,
		},
		{
			name:                  "Test 4 - Fail to cast payload",
			configApplyReturnsErr: fmt.Errorf("something went wrong"),
			message:               nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var rollbackRequiredError *RollbackRequiredError
			fakeFileManagerService := &filefakes.FakeFileManagerServiceInterface{}
			fakeFileManagerService.ConfigApplyReturns(test.configApplyReturnsErr)
			messagePipe := bus.NewFakeMessagePipe()
			filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection)
			err := filePlugin.Init(ctx, messagePipe)
			filePlugin.fileManagerService = fakeFileManagerService
			require.NoError(t, err)

			filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyRequestTopic, Data: test.message})

			messages := messagePipe.GetMessages()
			if test.configApplyReturnsErr == nil {
				assert.Equal(t, bus.WriteConfigSuccessfulTopic, messages[0].Topic)
				assert.Len(t, messages, 1)

				_, ok := messages[0].Data.(model.ConfigApplyMessage)
				assert.True(t, ok)
			} else if errors.As(test.configApplyReturnsErr, &rollbackRequiredError) {
				assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)
				assert.Len(t, messages, 2)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_ERROR,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
			} else {
				assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)
				assert.Len(t, messages, 1)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
			}
		})
	}
}

func TestFilePlugin_Process_ConfigUploadRequestTopic(t *testing.T) {
	ctx := context.Background()

	tempDir := os.TempDir()
	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	fileMeta := protos.FileMeta(testFile.Name(), "")

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

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
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

	fileMeta := protos.FileMeta("/unknown/file.conf", "")

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &mpi.ConfigUploadRequest{
				InstanceId: protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
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
	messagePipe := bus.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection)
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
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

func TestFilePlugin_Process_ConfigApplyFailedTopic(t *testing.T) {
	ctx := context.Background()
	instanceID := protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()

	tests := []struct {
		name            string
		rollbackReturns error
		instanceID      string
	}{
		{
			name:            "Test 1 - Rollback Success",
			rollbackReturns: nil,
			instanceID:      instanceID,
		},
		{
			name:            "Test 2 - Rollback Fail",
			rollbackReturns: fmt.Errorf("something went wrong"),
			instanceID:      instanceID,
		},

		{
			name:            "Test 3 - Fail to cast payload",
			rollbackReturns: fmt.Errorf("something went wrong"),
			instanceID:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockFileManager := &filefakes.FakeFileManagerServiceInterface{}
			mockFileManager.RollbackReturns(test.rollbackReturns)

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
			fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)

			messagePipe := bus.NewFakeMessagePipe()
			agentConfig := types.AgentConfig()
			filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection)

			err := filePlugin.Init(ctx, messagePipe)
			require.NoError(t, err)
			filePlugin.fileManagerService = mockFileManager

			data := model.ConfigApplyMessage{
				CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
				InstanceID:    test.instanceID,
			}

			filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyFailedTopic, Data: data})

			messages := messagePipe.GetMessages()

			if test.rollbackReturns == nil {
				assert.Equal(t, bus.RollbackWriteTopic, messages[0].Topic)
				assert.Len(t, messages, 1)
			} else {
				_, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				_, ok = messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Len(t, messages, 2)
			}
		})
	}
}
