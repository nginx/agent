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
	fileManagerService fileManagerServiceInterface
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
	case bus.ConnectionCreatedTopic:
		fp.fileManagerService.SetIsConnected(true)
	case bus.NginxConfigUpdateTopic:
		fp.handleNginxConfigUpdate(ctx, msg)
	case bus.ConfigUploadRequestTopic:
		fp.handleConfigUploadRequest(ctx, msg)
	case bus.ConfigApplyRequestTopic:
		fp.handleConfigApplyRequest(ctx, msg)
	case bus.ConfigApplySuccessfulTopic, bus.RollbackCompleteTopic:
		fp.clearCache()
	case bus.ConfigApplyFailedTopic:
		fp.handleConfigApplyFailedRequest(ctx, msg)
	default:
		slog.DebugContext(ctx, "File plugin unknown topic", "topic", msg.Topic)
	}
}

func (fp *FilePlugin) Subscriptions() []string {
	return []string{
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
		bus.ConfigApplyRequestTopic,
		bus.ConfigApplyFailedTopic,
		bus.ConfigApplySuccessfulTopic,
		bus.RollbackCompleteTopic,
	}
}

func (fp *FilePlugin) clearCache() {
	slog.Debug("Clearing cache after config apply")
	fp.fileManagerService.ClearCache()
}

func (fp *FilePlugin) handleConfigApplyFailedRequest(ctx context.Context, msg *bus.Message) {
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

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: applyResponse})
		fp.fileManagerService.ClearCache()

		return
	}

	// Send RollbackWriteTopic with Correlation and Instance ID for use by resource plugin
	fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.RollbackWriteTopic, Data: data})
}

func (fp *FilePlugin) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	var response *mpi.DataPlaneResponse
	correlationID := logger.GetCorrelationID(ctx)

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

	writeStatus, err := fp.fileManagerService.ConfigApply(ctx, configApplyRequest)

	switch writeStatus {
	case model.NoChange:
		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_OK,
			"Config apply successful, no files to change",
			configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			"",
		)

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
		fp.fileManagerService.ClearCache()

		return
	case model.Error:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes",
			"instance_id", configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			"error", err,
		)
		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed",
			configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			err.Error(),
		)

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
		fp.fileManagerService.ClearCache()

		return
	case model.RollbackRequired:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes, rolling back",
			"instance_id", configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			"error", err,
		)
		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config",
			configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			err.Error(),
		)

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})

		err := fp.fileManagerService.Rollback(ctx, configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId())
		if err != nil {
			// what to do?
			response = fp.createDataPlaneResponse(
				correlationID,
				mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				"Config apply failed, rollback failed",
				configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
				err.Error(),
			)
		}

		// what to do?
		response = fp.createDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed, rollback succeeded",
			configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
			"",
		)

		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: response})
		fp.fileManagerService.ClearCache()

		return
	case model.OK:
		// Send WriteConfigSuccessfulTopic with Correlation and Instance ID for use by resource plugin
		data := &model.ConfigApplyMessage{
			CorrelationID: correlationID,
			InstanceID:    configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId(),
		}
		fp.messagePipe.Process(ctx, &bus.Message{Topic: bus.WriteConfigSuccessfulTopic, Data: data})
	}
}

func (fp *FilePlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	err := fp.fileManagerService.UpdateOverview(ctx, nginxConfigContext.InstanceID, nginxConfigContext.Files)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to update file overview",
			"instance_id", nginxConfigContext.InstanceID,
			"error", err,
		)
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

func (fp *FilePlugin) createDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
	message, instanceID, err string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
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
