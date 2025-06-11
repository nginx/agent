// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nginx/agent/v3/pkg/id"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ bus.Plugin = (*ReadFilePlugin)(nil)

// The file plugin only writes, deletes and checks hashes of files
// the file plugin does not care about the instance type

type ReadFilePlugin struct {
	messagePipe        bus.MessagePipeInterface
	config             *config.Config
	conn               grpc.GrpcConnectionInterface
	fileManagerService fileManagerServiceInterface
}

func NewReadFilePlugin(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface) *ReadFilePlugin {
	return &ReadFilePlugin{
		config: agentConfig,
		conn:   grpcConnection,
	}
}

func (fp *ReadFilePlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting read only file plugin")

	fp.messagePipe = messagePipe
	fp.fileManagerService = NewFileManagerService(fp.conn.FileServiceClient(), fp.config)

	return nil
}

func (fp *ReadFilePlugin) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing read only file plugin")
	return fp.conn.Close(ctx)
}

func (fp *ReadFilePlugin) Info() *bus.Info {
	return &bus.Info{
		Name: "file",
	}
}

func (fp *ReadFilePlugin) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ConnectionResetTopic:
		if logger.ServerType(ctx) != "auxiliary" {

			return
		}
		fp.handleConnectionReset(ctx, msg)
	case bus.ConnectionCreatedTopic:
		if logger.ServerType(ctx) != "auxiliary" {
			return
		}
		fp.fileManagerService.SetIsConnected(true)
	case bus.NginxConfigUpdateTopic:
		fp.handleNginxConfigUpdate(ctx, msg)
	case bus.ConfigUploadRequestTopic:
		if logger.ServerType(ctx) != "auxiliary" {
			return
		}
		fp.handleConfigUploadRequest(ctx, msg)
	default:
		slog.DebugContext(ctx, "Read only file plugin unknown topic", "topic", msg.Topic)
	}
}

func (fp *ReadFilePlugin) Subscriptions() []string {
	return []string{
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
	}
}

func (fp *ReadFilePlugin) handleConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Read only file plugin received connection reset message")
	if newConnection, ok := msg.Data.(grpc.GrpcConnectionInterface); ok {
		var reconnect bool
		err := fp.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Read only file plugin unable to close connection", "error", err)
		}
		fp.conn = newConnection

		reconnect = fp.fileManagerService.IsConnected()
		fp.fileManagerService = NewFileManagerService(fp.conn.FileServiceClient(), fp.config)
		fp.fileManagerService.SetIsConnected(reconnect)

		slog.DebugContext(ctx, "Read only file plugin reset successfully")
	}
}

func (fp *ReadFilePlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	err := fp.fileManagerService.UpdateOverview(ctx, nginxConfigContext.InstanceID, nginxConfigContext.Files, 0)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", nginxConfigContext.InstanceID,
			"error", err,
		)
	}
}

func (fp *ReadFilePlugin) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
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
		err := fp.fileManagerService.UpdateFile(
			ctx,
			configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			file,
		)
		if err != nil {
			slog.ErrorContext(
				ctx,
				"Failed to update file",
				"instance_id", configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
				"file_name", file.GetFileMeta().GetName(),
				"error", err,
			)

			response := fp.createDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_ERROR,
				fmt.Sprintf("Failed to update file %s", file.GetFileMeta().GetName()),
				configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
				err.Error(),
			)

			updatingFilesError = err

			fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

			break
		}
	}

	response := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
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

func (fp *ReadFilePlugin) createDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
	message, instanceID, err string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  status,
			Message: message,
			Error:   err,
		},
		InstanceId: instanceID,
	}
}
