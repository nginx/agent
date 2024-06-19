// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bus.Plugin = (*FilePlugin)(nil)

// The file plugin only writes, deletes and checks hashes of files
// the file plugin does not care about the instance type

type FilePlugin struct {
	messagePipe        bus.MessagePipeInterface
	config             *config.Config
	conn               grpc.GrpcConnectionInterface
	fileManagerService *FileManagerService
}

func NewFilePlugin(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface) *FilePlugin {
	return &FilePlugin{
		config: agentConfig,
		conn:   grpcConnection,
	}
}

func (fp *FilePlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting file plugin")

	fp.messagePipe = messagePipe
	fp.fileManagerService = NewFileManagerService(fp.conn.FileServiceClient(), fp.config)

	return nil
}

func (fp *FilePlugin) Close(ctx context.Context) error {
	return fp.conn.Close(ctx)
}

func (fp *FilePlugin) Info() *bus.Info {
	return &bus.Info{
		Name: "file",
	}
}

func (fp *FilePlugin) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.NginxConfigUpdateTopic:
		fp.handleNginxConfigUpdate(ctx, msg)
	case bus.ConfigUploadRequestTopic:
		fp.handleConfigUploadRequest(ctx, msg)
	case bus.ConfigApplyRequestTopic:
		fp.handleConfigApplyRequest(ctx, msg)
	default:
		slog.DebugContext(ctx, "File plugin unknown topic", "topic", msg.Topic)
	}
}

func (fp *FilePlugin) Subscriptions() []string {
	return []string{
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
		bus.ConfigApplyRequestTopic,
	}
}

func (fp *FilePlugin) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	correlationID := logger.GetCorrelationID(ctx)

	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ConfigApplyRequest",
			"payload", msg.Data)
	}
	request, requestOk := managementPlaneRequest.GetRequest().(*mpi.ManagementPlaneRequest_ConfigApplyRequest)

	if !requestOk {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ConfigApplyRequest",
			"payload", msg.Data)

		response := &mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: correlationID,
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
				Message: "Config apply failed",
				Error:   "unable to cast to message payload",
			},
		}
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

		return
	}
	configApplyRequest := request.ConfigApplyRequest
	var response *mpi.DataPlaneResponse

	err := fp.fileManagerService.ConfigApply(ctx, configApplyRequest)

	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", configApplyRequest.GetConfigVersion().GetInstanceId(),
			"error", err,
		)

		response = &mpi.DataPlaneResponse{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: correlationID,
				Timestamp:     timestamppb.Now(),
			},
			CommandResponse: &mpi.CommandResponse{
				Status: mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: fmt.Sprintf("Config apply failed for instanceId: %s", configApplyRequest.
					GetConfigVersion().GetInstanceId()),
				Error: err.Error(),
			},
		}
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
	} else {
		// Send WriteConfigSuccessfulTopic with Correlation and Instance ID for use by resource plugin
		data := model.ConfigApply{
			CorrelationID: correlationID,
			InstanceID:    configApplyRequest.GetConfigVersion().GetInstanceId(),
		}
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.WriteConfigSuccessfulTopic, Data: data})
	}
}

func (fp *FilePlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)
	}

	err := fp.fileManagerService.UpdateOverview(ctx, nginxConfigContext.InstanceID, nginxConfigContext.Files)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", nginxConfigContext.InstanceID,
			"error", err,
		)

		return
	}
}

func (fp *FilePlugin) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(
			ctx,
			"Unable to cast message payload to *mpi.ManagementPlaneRequest",
			"payload", msg.Data,
		)

		return
	}

	configUploadRequest := managementPlaneRequest.GetConfigUploadRequest()

	correlationID := logger.GetCorrelationID(ctx)

	var updatingFilesError error

	for _, file := range configUploadRequest.GetOverview().GetFiles() {
		err := fp.fileManagerService.UpdateFile(ctx, configUploadRequest.GetInstanceId(), file)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"Failed to update file",
				"instance_id", configUploadRequest.GetInstanceId(),
				"file_name", file.GetFileMeta().GetName(),
				"error", err,
			)

			response := &mpi.DataPlaneResponse{
				MessageMeta: &mpi.MessageMeta{
					MessageId:     uuid.NewString(),
					CorrelationId: correlationID,
					Timestamp:     timestamppb.Now(),
				},
				CommandResponse: &mpi.CommandResponse{
					Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
					Message: fmt.Sprintf("Failed to update file %s", file.GetFileMeta().GetName()),
					Error:   err.Error(),
				},
			}

			updatingFilesError = err

			fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

			break
		}
	}

	response := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Successfully updated all files",
		},
	}

	if updatingFilesError != nil {
		response.CommandResponse.Status = mpi.CommandResponse_COMMAND_STATUS_FAILURE
		response.CommandResponse.Message = "Failed to update all files"
		response.CommandResponse.Error = updatingFilesError.Error()
	}

	fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
}
