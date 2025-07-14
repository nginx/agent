// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/bus/busfakes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/file/filefakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePlugin_Info(t *testing.T) {
	filePlugin := NewFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{},
		model.Command, &sync.RWMutex{})
	assert.Equal(t, "file", filePlugin.Info().Name)
}

func TestFilePlugin_Close(t *testing.T) {
	ctx := context.Background()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	filePlugin.Close(ctx)

	assert.Equal(t, 1, fakeGrpcConnection.CloseCallCount())
}

func TestFilePlugin_Subscriptions(t *testing.T) {
	filePlugin := NewFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{},
		model.Command, &sync.RWMutex{})
	assert.Equal(
		t,
		[]string{
			bus.ConnectionResetTopic,
			bus.ConnectionCreatedTopic,
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
			bus.ConfigApplyRequestTopic,
			bus.ConfigApplyFailedTopic,
			bus.ConfigApplySuccessfulTopic,
			bus.ConfigApplyCompleteTopic,
		},
		filePlugin.Subscriptions(),
	)

	readOnlyFilePlugin := NewFilePlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{},
		model.Auxiliary, &sync.RWMutex{})
	assert.Equal(t, []string{
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
	}, readOnlyFilePlugin.Subscriptions())
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
	fakeFileServiceClient.UpdateOverviewReturns(&mpi.UpdateOverviewResponse{
		Overview: nil,
	}, nil)

	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	fakeGrpcConnection.FileServiceClientReturns(fakeFileServiceClient)
	messagePipe := busfakes.NewFakeMessagePipe()

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
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

	filePath := tempDir + "/nginx.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	message := &mpi.ManagementPlaneRequest{
		Request: &mpi.ManagementPlaneRequest_ConfigApplyRequest{
			ConfigApplyRequest: protos.CreateConfigApplyRequest(protos.FileOverview(filePath, fileHash)),
		},
	}
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	tests := []struct {
		message               *mpi.ManagementPlaneRequest
		configApplyReturnsErr error
		name                  string
		configApplyStatus     model.WriteStatus
	}{
		{
			name:                  "Test 1 - Success",
			configApplyReturnsErr: nil,
			configApplyStatus:     model.OK,
			message:               message,
		},
		{
			name:                  "Test 2 - Fail, Rollback",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.RollbackRequired,
			message:               message,
		},
		{
			name:                  "Test 3 - Fail, No Rollback",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.Error,
			message:               message,
		},
		{
			name:                  "Test 4 - Fail to cast payload",
			configApplyReturnsErr: errors.New("something went wrong"),
			configApplyStatus:     model.Error,
			message:               nil,
		},
		{
			name:                  "Test 5 - No changes needed",
			configApplyReturnsErr: nil,
			configApplyStatus:     model.NoChange,
			message:               message,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeFileManagerService := &filefakes.FakeFileManagerServiceInterface{}
			fakeFileManagerService.ConfigApplyReturns(test.configApplyStatus, test.configApplyReturnsErr)
			messagePipe := busfakes.NewFakeMessagePipe()
			filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection, model.Command, &sync.RWMutex{})
			err := filePlugin.Init(ctx, messagePipe)
			filePlugin.fileManagerService = fakeFileManagerService
			require.NoError(t, err)

			filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyRequestTopic, Data: test.message})

			messages := messagePipe.Messages()

			switch {
			case test.configApplyStatus == model.OK:
				assert.Equal(t, bus.WriteConfigSuccessfulTopic, messages[0].Topic)
				assert.Len(t, messages, 1)

				_, ok := messages[0].Data.(*model.ConfigApplyMessage)
				assert.True(t, ok)
			case test.configApplyStatus == model.RollbackRequired:
				assert.Equal(t, bus.DataPlaneResponseTopic, messages[0].Topic)
				assert.Len(t, messages, 2)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_ERROR,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
				assert.Equal(t, "Config apply failed, rolling back config",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, test.configApplyReturnsErr.Error(), dataPlaneResponse.GetCommandResponse().GetError())
				dataPlaneResponse, ok = messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, "Config apply failed, rollback successful",
					dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus())
			case test.configApplyStatus == model.NoChange:
				assert.Len(t, messages, 1)

				response, ok := messages[0].Data.(*model.ConfigApplySuccess)
				assert.True(t, ok)
				assert.Equal(t, bus.ConfigApplySuccessfulTopic, messages[0].Topic)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_OK,
					response.DataPlaneResponse.GetCommandResponse().GetStatus(),
				)
			case test.message == nil:
				assert.Empty(t, messages)
			default:
				assert.Len(t, messages, 1)
				dataPlaneResponse, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(
					t,
					mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					dataPlaneResponse.GetCommandResponse().GetStatus(),
				)
				assert.Equal(t, "Config apply failed", dataPlaneResponse.GetCommandResponse().GetMessage())
				assert.Equal(t, test.configApplyReturnsErr.Error(), dataPlaneResponse.GetCommandResponse().GetError())
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

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
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

func TestFilePlugin_Process_ConfigUploadRequestTopic_Failure(t *testing.T) {
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

	filePlugin := NewFilePlugin(types.AgentConfig(), fakeGrpcConnection, model.Command, &sync.RWMutex{})
	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConnectionCreatedTopic})
	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})

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

func TestFilePlugin_Process_ConfigApplyFailedTopic(t *testing.T) {
	ctx := context.Background()
	instanceID := protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()

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
			rollbackReturns: errors.New("something went wrong"),
			instanceID:      instanceID,
		},

		{
			name:            "Test 3 - Fail to cast payload",
			rollbackReturns: errors.New("something went wrong"),
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

			messagePipe := busfakes.NewFakeMessagePipe()
			agentConfig := types.AgentConfig()
			filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection, model.Command, &sync.RWMutex{})

			err := filePlugin.Init(ctx, messagePipe)
			require.NoError(t, err)
			filePlugin.fileManagerService = mockFileManager

			data := &model.ConfigApplyMessage{
				CorrelationID: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
				InstanceID:    test.instanceID,
				Error:         errors.New("something went wrong with config apply"),
			}

			filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyFailedTopic, Data: data})

			messages := messagePipe.Messages()

			switch {
			case test.rollbackReturns == nil:
				assert.Equal(t, bus.RollbackWriteTopic, messages[0].Topic)
				assert.Len(t, messages, 1)

			case test.instanceID == "":
				assert.Empty(t, messages)
			default:
				rollbackMessage, ok := messages[0].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, "Rollback failed", rollbackMessage.GetCommandResponse().GetMessage())
				assert.Equal(t, test.rollbackReturns.Error(), rollbackMessage.GetCommandResponse().GetError())
				applyMessage, ok := messages[1].Data.(*mpi.DataPlaneResponse)
				assert.True(t, ok)
				assert.Equal(t, "Config apply failed, rollback failed",
					applyMessage.GetCommandResponse().GetMessage())
				assert.Equal(t, data.Error.Error(), applyMessage.GetCommandResponse().GetError())
				assert.Len(t, messages, 2)
			}
		})
	}
}

func TestFilePlugin_Process_ConfigApplyRollbackCompleteTopic(t *testing.T) {
	ctx := context.Background()
	instance := protos.NginxOssInstance([]string{})
	mockFileManager := &filefakes.FakeFileManagerServiceInterface{}

	messagePipe := busfakes.NewFakeMessagePipe()
	agentConfig := types.AgentConfig()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection, model.Command, &sync.RWMutex{})

	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	filePlugin.fileManagerService = mockFileManager

	expectedResponse := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Config apply successful",
			Error:   "",
		},
		InstanceId: instance.GetInstanceMeta().GetInstanceId(),
	}

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplySuccessfulTopic, Data: &model.ConfigApplySuccess{
		ConfigContext:     &model.NginxConfigContext{},
		DataPlaneResponse: expectedResponse,
	}})

	messages := messagePipe.Messages()
	response, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)

	assert.Equal(t, expectedResponse.GetCommandResponse().GetStatus(), response.GetCommandResponse().GetStatus())
	assert.Equal(t, expectedResponse.GetCommandResponse().GetMessage(), response.GetCommandResponse().GetMessage())
	assert.Equal(t, expectedResponse.GetCommandResponse().GetError(), response.GetCommandResponse().GetError())
	assert.Equal(t, expectedResponse.GetMessageMeta().GetCorrelationId(), response.GetMessageMeta().GetCorrelationId())

	assert.Equal(t, expectedResponse.GetInstanceId(), response.GetInstanceId())
}

func TestFilePlugin_Process_ConfigApplyCompleteTopic(t *testing.T) {
	ctx := context.Background()
	instance := protos.NginxOssInstance([]string{})
	mockFileManager := &filefakes.FakeFileManagerServiceInterface{}

	messagePipe := busfakes.NewFakeMessagePipe()
	agentConfig := types.AgentConfig()
	fakeGrpcConnection := &grpcfakes.FakeGrpcConnectionInterface{}
	filePlugin := NewFilePlugin(agentConfig, fakeGrpcConnection, model.Command, &sync.RWMutex{})

	err := filePlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	filePlugin.fileManagerService = mockFileManager
	expectedResponse := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Config apply successful",
			Error:   "",
		},
		InstanceId: instance.GetInstanceMeta().GetInstanceId(),
	}

	filePlugin.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: expectedResponse})

	messages := messagePipe.Messages()
	response, ok := messages[0].Data.(*mpi.DataPlaneResponse)
	assert.True(t, ok)

	assert.Equal(t, expectedResponse.GetCommandResponse().GetStatus(), response.GetCommandResponse().GetStatus())
	assert.Equal(t, expectedResponse.GetCommandResponse().GetMessage(), response.GetCommandResponse().GetMessage())
	assert.Equal(t, expectedResponse.GetCommandResponse().GetError(), response.GetCommandResponse().GetError())
	assert.Equal(t, expectedResponse.GetMessageMeta().GetCorrelationId(), response.GetMessageMeta().GetCorrelationId())

	assert.Equal(t, expectedResponse.GetInstanceId(), response.GetInstanceId())
}
