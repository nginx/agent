// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"log/slog"
	"sync"

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
		UpdateClient(client mpi.CommandServiceClient)
		Resource() *mpi.Resource
		Subscribe(ctx context.Context)
		IsConnected() bool
		CreateConnection(ctx context.Context, resource *mpi.Resource) (*mpi.CreateConnectionResponse, error)
	}

	CommandPlugin struct {
		messagePipe      bus.MessagePipeInterface
		config           *config.Config
		subscribeCancel  context.CancelFunc
		conn             grpc.GrpcConnectionInterface
		commandService   commandService
		subscribeChannel chan *mpi.ManagementPlaneRequest
		subscribeMutex   sync.Mutex
	}
)

func NewCommandPlugin(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface) *CommandPlugin {
	return &CommandPlugin{
		config:           agentConfig,
		conn:             grpcConnection,
		subscribeChannel: make(chan *mpi.ManagementPlaneRequest),
	}
}

func (cp *CommandPlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting command plugin")

	cp.messagePipe = messagePipe
	cp.commandService = NewCommandService(cp.conn.CommandServiceClient(), cp.config, cp.subscribeChannel)

	go cp.monitorSubscribeChannel(ctx)

	return nil
}

func (cp *CommandPlugin) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Canceling subscribe context")

	cp.subscribeMutex.Lock()
	if cp.subscribeCancel != nil {
		cp.subscribeCancel()
	}
	cp.subscribeMutex.Unlock()

	return cp.conn.Close(ctx)
}

func (cp *CommandPlugin) Info() *bus.Info {
	return &bus.Info{
		Name: "command",
	}
}

func (cp *CommandPlugin) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ConnectionResetTopic:
		cp.processConnectionReset(ctx, msg)
	case bus.ResourceUpdateTopic:
		cp.processResourceUpdate(ctx, msg)
	case bus.InstanceHealthTopic:
		cp.processInstanceHealth(ctx, msg)
	case bus.DataPlaneHealthResponseTopic:
		cp.processDataPlaneHealth(ctx, msg)
	case bus.DataPlaneResponseTopic:
		cp.processDataPlaneResponse(ctx, msg)
	default:
		slog.DebugContext(ctx, "Command plugin unknown topic", "topic", msg.Topic)
	}
}

func (cp *CommandPlugin) processResourceUpdate(ctx context.Context, msg *bus.Message) {
	if resource, ok := msg.Data.(*mpi.Resource); ok {
		if !cp.commandService.IsConnected() && cp.config.IsFeatureEnabled(pkgConfig.FeatureConnection) {
			cp.createConnection(ctx, resource)
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
	if instances, ok := msg.Data.([]*mpi.InstanceHealth); ok {
		err := cp.commandService.UpdateDataPlaneHealth(ctx, instances)
		correlationID := logger.GetCorrelationID(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to update data plane health", "error", err)
			cp.messagePipe.Process(ctx, &bus.Message{
				Topic: bus.DataPlaneResponseTopic,
				Data: cp.createDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					"Failed to send the health status update", err.Error()),
			})
		}
		cp.messagePipe.Process(ctx, &bus.Message{
			Topic: bus.DataPlaneResponseTopic,
			Data: cp.createDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
				"Successfully sent the health status update", ""),
		})
	}
}

func (cp *CommandPlugin) processInstanceHealth(ctx context.Context, msg *bus.Message) {
	if instances, ok := msg.Data.([]*mpi.InstanceHealth); ok {
		err := cp.commandService.UpdateDataPlaneHealth(ctx, instances)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to update data plane health", "error", err)
		}
	}
}

func (cp *CommandPlugin) processDataPlaneResponse(ctx context.Context, msg *bus.Message) {
	if response, ok := msg.Data.(*mpi.DataPlaneResponse); ok {
		err := cp.commandService.SendDataPlaneResponse(ctx, response)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to send data plane response", "error", err)
		}
	}
}

func (cp *CommandPlugin) processConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Command plugin received connection reset")
	if newConnection, ok := msg.Data.(*grpc.GrpcConnection); ok {
		if !cp.commandService.IsConnected() {
			slog.DebugContext(ctx, "Command plugin: service is not connected")
			return
		}
		err := cp.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to close connection", "error", err)
		}
		cp.conn = newConnection
		cp.commandService.UpdateClient(cp.conn.CommandServiceClient())
		//go cp.monitorSubscribeChannel(ctx)
		//cp.createConnection(ctx, cp.commandService.Resource())
		slog.DebugContext(ctx, "Command plugin: client reset successfully")
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
				cp.handleConfigUploadRequest(newCtx, message)
			case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
				cp.handleConfigApplyRequest(newCtx, message)
			case *mpi.ManagementPlaneRequest_HealthRequest:
				cp.handleHealthRequest(newCtx)
			case *mpi.ManagementPlaneRequest_ActionRequest:
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
			"API Action Request feature disabled. Unable to process API action request",
			"request", message,
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
			"request", message,
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
			"request", message,
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
