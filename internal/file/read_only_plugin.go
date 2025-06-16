// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/command"

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

func (rp *ReadFilePlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting read file plugin")

	rp.messagePipe = messagePipe
	rp.fileManagerService = NewFileManagerService(rp.conn.FileServiceClient(), rp.config)

	return nil
}

func (rp *ReadFilePlugin) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing read file plugin")
	return rp.conn.Close(ctx)
}

func (rp *ReadFilePlugin) Info() *bus.Info {
	return &bus.Info{
		Name: "read",
	}
}

func (rp *ReadFilePlugin) Process(ctx context.Context, msg *bus.Message) {
	if logger.ServerType(ctx) == command.Auxiliary.String() || logger.ServerType(ctx) == "" {
		switch msg.Topic {
		case bus.ConnectionResetTopic:
			rp.handleConnectionReset(ctx, msg)
		case bus.ConnectionCreatedTopic:
			slog.InfoContext(ctx, "Read file plugin received connection created message")
			rp.fileManagerService.SetIsConnected(true)
		case bus.NginxConfigUpdateTopic:
			rp.handleNginxConfigUpdate(ctx, msg)
		case bus.ConfigUploadRequestTopic:
			rp.handleConfigUploadRequest(ctx, msg)
		default:
			slog.DebugContext(ctx, "Read file plugin received unknown topic", "topic", msg.Topic)
		}
	}
}

func (rp *ReadFilePlugin) Subscriptions() []string {
	return []string{
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
	}
}

func (rp *ReadFilePlugin) handleConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Read file plugin received connection reset message")
	if newConnection, ok := msg.Data.(grpc.GrpcConnectionInterface); ok {
		var reconnect bool
		err := rp.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Read file plugin unable to close connection", "error", err)
		}
		rp.conn = newConnection

		reconnect = rp.fileManagerService.IsConnected()
		rp.fileManagerService = NewFileManagerService(rp.conn.FileServiceClient(), rp.config)
		rp.fileManagerService.SetIsConnected(reconnect)

		slog.DebugContext(ctx, "File manager service client reset successfully")
	}
}

func (rp *ReadFilePlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Read file plugin received nginx config update message")
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	rp.fileManagerService.ConfigUpdate(ctx, nginxConfigContext)
}

// nolint: dupl
func (rp *ReadFilePlugin) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Read file plugin received config upload request message")
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

	correlationID := logger.CorrelationID(ctx)

	updatingFilesError := rp.fileManagerService.ConfigUpload(ctx, configUploadRequest)

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

	rp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
}
