// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	pkgConfig "github.com/nginx/agent/v3/pkg/config"
	"github.com/nginx/agent/v3/pkg/id"
)

var _ bus.Plugin = (*CommandPlugin)(nil)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . commandService

type (
	commandService interface {
		UpdateDataPlaneStatus(ctx context.Context, resource *mpi.Resource) error
		UpdateDataPlaneHealth(ctx context.Context, instanceHealths []*mpi.InstanceHealth) error
		SendDataPlaneResponse(ctx context.Context, response *mpi.DataPlaneResponse) error
		UpdateClient(ctx context.Context, client mpi.CommandServiceClient) error
		Subscribe(ctx context.Context)
		IsConnected() bool
		CreateConnection(ctx context.Context, resource *mpi.Resource) (*mpi.CreateConnectionResponse, error)
	}

	CommandPlugin struct {
		messagePipe       bus.MessagePipeInterface
		config            *config.Config
		subscribeCancel   context.CancelFunc
		conn              grpc.GrpcConnectionInterface
		commandService    commandService
		subscribeChannel  chan *mpi.ManagementPlaneRequest
		commandServerType model.ServerType
		subscribeMutex    sync.Mutex
	}
)

func NewCommandPlugin(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface,
	commandServerType model.ServerType,
) *CommandPlugin {
	return &CommandPlugin{
		config:            agentConfig,
		conn:              grpcConnection,
		subscribeChannel:  make(chan *mpi.ManagementPlaneRequest),
		commandServerType: commandServerType,
	}
}

func (cp *CommandPlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	newCtx := context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, cp.commandServerType.String()),
	)
	slog.DebugContext(newCtx, "Starting command plugin")

	cp.messagePipe = messagePipe
	cp.commandService = NewCommandService(cp.conn.CommandServiceClient(), cp.config, cp.subscribeChannel)

	go cp.monitorSubscribeChannel(newCtx)

	return nil
}

func (cp *CommandPlugin) Close(ctx context.Context) error {
	newCtx := context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, cp.commandServerType.String()),
	)
	slog.InfoContext(newCtx, "Closing command plugin")

	cp.subscribeMutex.Lock()
	if cp.subscribeCancel != nil {
		cp.subscribeCancel()
	}
	cp.subscribeMutex.Unlock()

	return cp.conn.Close(newCtx)
}

func (cp *CommandPlugin) Info() *bus.Info {
	name := "command"
	if cp.commandServerType.String() == model.Auxiliary.String() {
		name = "auxiliary-command"
	}

	return &bus.Info{
		Name: name,
	}
}

func (cp *CommandPlugin) Process(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Processing command")
	ctxWithMetadata := cp.config.NewContextWithLabels(ctx)

	if logger.ServerType(ctxWithMetadata) == "" {
		ctxWithMetadata = context.WithValue(
			ctxWithMetadata,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, cp.commandServerType.String()),
		)
	}

	if logger.ServerType(ctxWithMetadata) == cp.commandServerType.String() {
		switch msg.Topic {
		case bus.ConnectionResetTopic:
			cp.processConnectionReset(ctxWithMetadata, msg)
		case bus.ResourceUpdateTopic:
			cp.processResourceUpdate(ctxWithMetadata, msg)
		case bus.InstanceHealthTopic:
			cp.processInstanceHealth(ctxWithMetadata, msg)
		case bus.DataPlaneHealthResponseTopic:
			cp.processDataPlaneHealth(ctxWithMetadata, msg)
		case bus.DataPlaneResponseTopic:
			cp.processDataPlaneResponse(ctxWithMetadata, msg)
		default:
			slog.DebugContext(ctxWithMetadata, "Command plugin received unknown topic", "topic", msg.Topic)
		}
	}
}

func (cp *CommandPlugin) Subscriptions() []string {
	return []string{
		bus.ConnectionResetTopic,
		bus.ResourceUpdateTopic,
		bus.InstanceHealthTopic,
		bus.DataPlaneHealthResponseTopic,
		bus.DataPlaneResponseTopic,
	}
}

func (cp *CommandPlugin) processResourceUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received resource update message")
	if resource, ok := msg.Data.(*mpi.Resource); ok {
		if !cp.commandService.IsConnected() {
			newCtx := context.WithValue(
				ctx,
				logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, cp.commandServerType.String()),
			)
			cp.createConnection(newCtx, resource)
		} else {
			statusErr := cp.commandService.UpdateDataPlaneStatus(ctx, resource)
			if statusErr != nil {
				slog.ErrorContext(ctx, "Unable to update data plane status", "error", statusErr)
			}
		}
	}
}

func (cp *CommandPlugin) createConnection(ctx context.Context, resource *mpi.Resource) {
	var subscribeCtx context.Context

	createConnectionResponse, err := cp.commandService.CreateConnection(ctx, resource)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to create connection", "error", err)
	}

	if createConnectionResponse != nil {
		cp.subscribeMutex.Lock()
		subscribeCtx, cp.subscribeCancel = context.WithCancel(ctx)
		cp.subscribeMutex.Unlock()

		go cp.commandService.Subscribe(subscribeCtx)

		cp.messagePipe.Process(ctx, &bus.Message{
			Topic: bus.ConnectionCreatedTopic,
			Data:  createConnectionResponse,
		})
	}
}

func (cp *CommandPlugin) processDataPlaneHealth(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received data plane health message")
	if instances, ok := msg.Data.([]*mpi.InstanceHealth); ok {
		err := cp.commandService.UpdateDataPlaneHealth(ctx, instances)
		correlationID := logger.CorrelationID(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to update data plane health", "error", err)

			cp.processDataPlaneResponse(ctx, &bus.Message{
				Topic: bus.DataPlaneResponseTopic,
				Data: cp.createDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					"Failed to send the health status update", err.Error()),
			})
		}
		cp.processDataPlaneResponse(ctx, &bus.Message{
			Topic: bus.DataPlaneResponseTopic,
			Data: cp.createDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
				"Successfully sent health status update", ""),
		})
	}
}

func (cp *CommandPlugin) processInstanceHealth(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received instance health message")
	if instances, ok := msg.Data.([]*mpi.InstanceHealth); ok {
		err := cp.commandService.UpdateDataPlaneHealth(ctx, instances)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to update data plane health", "error", err)
		}
	}
}

func (cp *CommandPlugin) processDataPlaneResponse(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received data plane response message")
	if response, ok := msg.Data.(*mpi.DataPlaneResponse); ok {
		slog.InfoContext(ctx, "Sending data plane response message", "message",
			response.GetCommandResponse().GetMessage(), "status", response.GetCommandResponse().GetStatus())
		err := cp.commandService.SendDataPlaneResponse(ctx, response)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to send data plane response", "error", err)
		}
	}
}

func (cp *CommandPlugin) processConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received connection reset message")
	if newConnection, ok := msg.Data.(grpc.GrpcConnectionInterface); ok {
		connectionErr := cp.conn.Close(ctx)
		if connectionErr != nil {
			slog.ErrorContext(ctx, "Command plugin: unable to close connection", "error", connectionErr)
		}
		cp.conn = newConnection
		err := cp.commandService.UpdateClient(ctx, cp.conn.CommandServiceClient())
		if err != nil {
			slog.ErrorContext(ctx, "Failed to reset connection", "error", err)
			return
		}
		slog.DebugContext(ctx, "Command service client reset successfully")
	}
}

// nolint: revive, cyclop
func (cp *CommandPlugin) monitorSubscribeChannel(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-cp.subscribeChannel:
			newCtx := context.WithValue(
				ctx,
				logger.CorrelationIDContextKey,
				slog.Any(logger.CorrelationIDKey, message.GetMessageMeta().GetCorrelationId()),
			)
			slog.DebugContext(newCtx, "Received management plane request", "request", message)

			switch message.GetRequest().(type) {
			case *mpi.ManagementPlaneRequest_ConfigUploadRequest:
				slog.InfoContext(ctx, "Received management plane config upload request")
				cp.handleConfigUploadRequest(newCtx, message)
			case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
				if cp.commandServerType != model.Command {
					slog.WarnContext(newCtx, "Auxiliary command server can not perform config apply",
						"command_server_type", cp.commandServerType.String())
					cp.handleInvalidRequest(newCtx, message, "Config apply failed",
						message.GetConfigApplyRequest().GetOverview().GetConfigVersion().GetInstanceId())

					return
				}
				slog.InfoContext(ctx, "Received management plane config apply request")
				cp.handleConfigApplyRequest(newCtx, message)
			case *mpi.ManagementPlaneRequest_HealthRequest:
				slog.InfoContext(ctx, "Received management plane health request")
				cp.handleHealthRequest(newCtx)
			case *mpi.ManagementPlaneRequest_ActionRequest:
				if cp.commandServerType != model.Command {
					slog.WarnContext(newCtx, "Auxiliary command server can not perform api action",
						"command_server_type", cp.commandServerType.String())
					cp.handleInvalidRequest(newCtx, message, "API action failed",
						message.GetActionRequest().GetInstanceId())

					return
				}
				slog.InfoContext(ctx, "Received management plane action request")
				cp.handleAPIActionRequest(newCtx, message)
			default:
				slog.DebugContext(newCtx, "Management plane request not implemented yet")
			}
		}
	}
}

func (cp *CommandPlugin) handleAPIActionRequest(ctx context.Context, message *mpi.ManagementPlaneRequest) {
	if cp.config.IsFeatureEnabled(pkgConfig.FeatureAPIAction) {
		cp.messagePipe.Process(ctx, &bus.Message{Topic: bus.APIActionRequestTopic, Data: message})
	} else {
		slog.WarnContext(
			ctx,
			"API action feature disabled. Unable to process API action request",
			"request", message, "enabled_features", cp.config.Features,
		)

		err := cp.commandService.SendDataPlaneResponse(ctx, &mpi.DataPlaneResponse{
			MessageMeta: message.GetMessageMeta(),
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "API action failed",
				Error:   "API action feature is disabled",
			},
			InstanceId: message.GetActionRequest().GetInstanceId(),
		})
		if err != nil {
			slog.ErrorContext(ctx, "Unable to send data plane response", "error", err)
		}
	}
}

func (cp *CommandPlugin) handleConfigApplyRequest(newCtx context.Context, message *mpi.ManagementPlaneRequest) {
	if cp.config.IsFeatureEnabled(pkgConfig.FeatureConfiguration) {
		cp.messagePipe.Process(newCtx, &bus.Message{Topic: bus.ConfigApplyRequestTopic, Data: message})
	} else {
		slog.WarnContext(
			newCtx,
			"Configuration feature disabled. Unable to process config apply request",
			"request", message, "enabled_features", cp.config.Features,
		)

		err := cp.commandService.SendDataPlaneResponse(newCtx, &mpi.DataPlaneResponse{
			MessageMeta: message.GetMessageMeta(),
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config apply failed",
				Error:   "Configuration feature is disabled",
			},
			InstanceId: message.GetConfigApplyRequest().GetOverview().GetConfigVersion().GetInstanceId(),
		})
		if err != nil {
			slog.ErrorContext(newCtx, "Unable to send data plane response", "error", err)
		}
	}
}

func (cp *CommandPlugin) handleConfigUploadRequest(newCtx context.Context, message *mpi.ManagementPlaneRequest) {
	if cp.config.IsFeatureEnabled(pkgConfig.FeatureConfiguration) {
		cp.messagePipe.Process(newCtx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})
	} else {
		slog.WarnContext(
			newCtx,
			"Configuration feature disabled. Unable to process config upload request",
			"request", message, "enabled_features", cp.config.Features,
		)

		err := cp.commandService.SendDataPlaneResponse(newCtx, &mpi.DataPlaneResponse{
			MessageMeta: message.GetMessageMeta(),
			CommandResponse: &mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config upload failed",
				Error:   "Configuration feature is disabled",
			},
			InstanceId: message.GetConfigUploadRequest().GetOverview().GetConfigVersion().GetInstanceId(),
		})
		if err != nil {
			slog.ErrorContext(newCtx, "Unable to send data plane response", "error", err)
		}
	}
}

func (cp *CommandPlugin) handleHealthRequest(newCtx context.Context) {
	cp.messagePipe.Process(newCtx, &bus.Message{Topic: bus.DataPlaneHealthRequestTopic})
}

func (cp *CommandPlugin) handleInvalidRequest(ctx context.Context,
	request *mpi.ManagementPlaneRequest, message, instanceID string,
) {
	err := cp.commandService.SendDataPlaneResponse(ctx, &mpi.DataPlaneResponse{
		MessageMeta: request.GetMessageMeta(),
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			Message: message,
			Error:   "Unable to process request. Management plane is configured as read only.",
		},
		InstanceId: instanceID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "Unable to send data plane response", "error", err)
	}
}

func (cp *CommandPlugin) createDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
	message, err string,
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
	}
}
