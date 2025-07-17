// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"

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
	manifestLock       *sync.RWMutex
	messagePipe        bus.MessagePipeInterface
	config             *config.Config
	conn               grpc.GrpcConnectionInterface
	fileManagerService fileManagerServiceInterface
	serverType         model.ServerType
}

func NewFilePlugin(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface,
	serverType model.ServerType, manifestLock *sync.RWMutex,
) *FilePlugin {
	return &FilePlugin{
		config:       agentConfig,
		conn:         grpcConnection,
		serverType:   serverType,
		manifestLock: manifestLock,
	}
}

func (fp *FilePlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, fp.serverType.String()),
	)
	slog.DebugContext(ctx, "Starting file plugin")

	fp.messagePipe = messagePipe
	fp.fileManagerService = NewFileManagerService(fp.conn.FileServiceClient(), fp.config, fp.manifestLock)

	return nil
}

func (fp *FilePlugin) Close(ctx context.Context) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, fp.serverType.String()),
	)
	slog.InfoContext(ctx, "Closing file plugin")

	return fp.conn.Close(ctx)
}

func (fp *FilePlugin) Info() *bus.Info {
	name := "file"
	if fp.serverType.String() == model.Auxiliary.String() {
		name = "auxiliary-file"
	}

	return &bus.Info{
		Name: name,
	}
}

// nolint: cyclop, revive
func (fp *FilePlugin) Process(ctx context.Context, msg *bus.Message) {
	ctxWithMetadata := fp.config.NewContextWithLabels(ctx)

	if logger.ServerType(ctx) == "" {
		ctxWithMetadata = context.WithValue(
			ctxWithMetadata,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, fp.serverType.String()),
		)
	}

	if logger.ServerType(ctxWithMetadata) == fp.serverType.String() {
		switch msg.Topic {
		case bus.ConnectionResetTopic:
			fp.handleConnectionReset(ctxWithMetadata, msg)
		case bus.ConnectionCreatedTopic:
			slog.DebugContext(ctxWithMetadata, "File plugin received connection created message")
			fp.fileManagerService.SetIsConnected(true)
		case bus.NginxConfigUpdateTopic:
			fp.handleNginxConfigUpdate(ctxWithMetadata, msg)
		case bus.ConfigUploadRequestTopic:
			fp.handleConfigUploadRequest(ctxWithMetadata, msg)
		case bus.ConfigApplyRequestTopic:
			fp.handleConfigApplyRequest(ctxWithMetadata, msg)
		case bus.ConfigApplyCompleteTopic:
			fp.handleConfigApplyComplete(ctxWithMetadata, msg)
		case bus.ConfigApplySuccessfulTopic:
			fp.handleConfigApplySuccess(ctxWithMetadata, msg)
		case bus.ConfigApplyFailedTopic:
			fp.handleConfigApplyFailedRequest(ctxWithMetadata, msg)
		default:
			slog.DebugContext(ctxWithMetadata, "File plugin received unknown topic", "topic", msg.Topic)
		}
	}
}

func (fp *FilePlugin) Subscriptions() []string {
	if fp.serverType == model.Auxiliary {
		return []string{
			bus.ConnectionResetTopic,
			bus.ConnectionCreatedTopic,
			bus.NginxConfigUpdateTopic,
			bus.ConfigUploadRequestTopic,
		}
	}

	return []string{
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
		bus.ConfigApplyRequestTopic,
		bus.ConfigApplyFailedTopic,
		bus.ConfigApplySuccessfulTopic,
		bus.ConfigApplyCompleteTopic,
	}
}

func (fp *FilePlugin) handleConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received connection reset message")
	if newConnection, ok := msg.Data.(grpc.GrpcConnectionInterface); ok {
		var reconnect bool
		err := fp.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "File plugin: unable to close connection", "error", err)
		}
		fp.conn = newConnection

		reconnect = fp.fileManagerService.IsConnected()
		fp.fileManagerService = NewFileManagerService(fp.conn.FileServiceClient(), fp.config, fp.manifestLock)
		fp.fileManagerService.SetIsConnected(reconnect)

		slog.DebugContext(ctx, "File manager service client reset successfully")
	}
}

func (fp *FilePlugin) handleConfigApplyComplete(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received config apply complete message")
	response, ok := msg.Data.(*mpi.DataPlaneResponse)

	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.DataPlaneResponse", "payload", msg.Data)
		return
	}

	fp.fileManagerService.ClearCache()
	fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
}

func (fp *FilePlugin) handleConfigApplySuccess(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received config success message")
	successMessage, ok := msg.Data.(*model.ConfigApplySuccess)

	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplySuccess", "payload", msg.Data)
		return
	}

	fp.fileManagerService.ClearCache()

	if successMessage.ConfigContext.Files != nil {
		slog.DebugContext(ctx, "Changes made during config apply, update files on disk")
		updateError := fp.fileManagerService.UpdateCurrentFilesOnDisk(
			ctx,
			files.ConvertToMapOfFiles(successMessage.ConfigContext.Files),
			true,
		)
		if updateError != nil {
			slog.ErrorContext(ctx, "Unable to update current files on disk", "error", updateError)
		}
	}
	fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: successMessage.DataPlaneResponse})
}

func (fp *FilePlugin) handleConfigApplyFailedRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received config failed message")

	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if data.InstanceID == "" || !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage",
			"payload", msg.Data)
		fp.fileManagerService.ClearCache()

		return
	}

	err := fp.fileManagerService.Rollback(ctx, data.InstanceID)
	if err != nil {
		rollbackResponse := fp.createDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Rollback failed", data.InstanceID, err.Error())

		applyResponse := fp.createDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed, rollback failed", data.InstanceID, data.Error.Error())

		fp.fileManagerService.ClearCache()
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: applyResponse})

		return
	}

	// Send RollbackWriteTopic with Correlation and Instance ID for use by resource plugin
	fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.RollbackWriteTopic, Data: data})
}

func (fp *FilePlugin) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received config apply request message")
	var response *mpi.DataPlaneResponse
	correlationID := logger.CorrelationID(ctx)

	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest",
			"payload", msg.Data)

		return
	}

	request, requestOk := managementPlaneRequest.GetRequest().(*mpi.ManagementPlaneRequest_ConfigApplyRequest)
	if !requestOk {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ConfigApplyRequest",
			"payload", msg.Data)

		return
	}

	configApplyRequest := request.ConfigApplyRequest
	instanceID := configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId()

	writeStatus, err := fp.fileManagerService.ConfigApply(ctx, configApplyRequest)

	switch writeStatus {
	case model.NoChange:
		slog.DebugContext(ctx, "No changes required for config apply request")
		dpResponse := fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_OK,
			"Config apply successful, no files to change",
			instanceID,
			"",
		)

		successMessage := &model.ConfigApplySuccess{
			ConfigContext:     &model.NginxConfigContext{},
			DataPlaneResponse: dpResponse,
		}

		fp.fileManagerService.ClearCache()
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplySuccessfulTopic, Data: successMessage})

		return
	case model.Error:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes",
			"instance_id", instanceID,
			"error", err,
		)
		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed",
			instanceID,
			err.Error(),
		)

		fp.fileManagerService.ClearCache()
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: response})

		return
	case model.RollbackRequired:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes, rolling back",
			"instance_id", instanceID,
			"error", err,
		)

		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config",
			instanceID,
			err.Error(),
		)
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

		rollbackErr := fp.fileManagerService.Rollback(
			ctx,
			instanceID,
		)
		if rollbackErr != nil {
			rollbackResponse := fp.createDataPlaneResponse(
				correlationID,
				mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				"Config apply failed, rollback failed",
				instanceID,
				rollbackErr.Error())

			fp.fileManagerService.ClearCache()
			fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: rollbackResponse})

			return
		}

		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed, rollback successful",
			instanceID,
			err.Error())

		fp.fileManagerService.ClearCache()
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: response})

		return
	case model.OK:
		slog.DebugContext(ctx, "Changes required for config apply request")
		// Send WriteConfigSuccessfulTopic with Correlation and Instance ID for use by resource plugin
		data := &model.ConfigApplyMessage{
			CorrelationID: correlationID,
			InstanceID:    instanceID,
		}

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.WriteConfigSuccessfulTopic, Data: data})
	}
}

func (fp *FilePlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received nginx config update message")
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	fp.fileManagerService.ConfigUpdate(ctx, nginxConfigContext)
}

func (fp *FilePlugin) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "File plugin received config upload request message")
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

	updatingFilesError := fp.fileManagerService.ConfigUpload(ctx, configUploadRequest)

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

func (fp *FilePlugin) createDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
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
