// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"context"
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
)

var _ bus.Plugin = (*CommandPlugin)(nil)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . commandService

type (
	commandService interface {
		UpdateDataPlaneStatus(ctx context.Context, resource *mpi.Resource) error
		UpdateDataPlaneHealth(ctx context.Context, instanceHealths []*mpi.InstanceHealth) error
		SendDataPlaneResponse(ctx context.Context, response *mpi.DataPlaneResponse) error
		CancelSubscription(ctx context.Context)
	}

	CommandPlugin struct {
		messagePipe      bus.MessagePipeInterface
		config           *config.Config
		conn             grpc.GrpcConnectionInterface
		commandService   commandService
		subscribeChannel chan *mpi.ManagementPlaneRequest
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
	cp.commandService = NewCommandService(ctx, cp.conn.CommandServiceClient(), cp.config, cp.subscribeChannel)

	go cp.monitorSubscribeChannel(ctx)

	return nil
}

func (cp *CommandPlugin) Close(ctx context.Context) error {
	cp.commandService.CancelSubscription(ctx)
	return cp.conn.Close(ctx)
}

func (cp *CommandPlugin) Info() *bus.Info {
	return &bus.Info{
		Name: "command",
	}
}

func (cp *CommandPlugin) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ResourceUpdateTopic:
		cp.processResourceUpdate(ctx, msg)
	case bus.InstanceHealthTopic:
		cp.processInstanceHealth(ctx, msg)
	case bus.DataPlaneResponseTopic:
		cp.processDataPlaneResponse(ctx, msg)
	default:
		slog.DebugContext(ctx, "Command plugin unknown topic", "topic", msg.Topic)
	}
}

func (cp *CommandPlugin) processResourceUpdate(ctx context.Context, msg *bus.Message) {
	if resource, ok := msg.Data.(*mpi.Resource); ok {
		err := cp.commandService.UpdateDataPlaneStatus(ctx, resource)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to update data plane status", "error", err)
		}
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

func (cp *CommandPlugin) Subscriptions() []string {
	return []string{
		bus.ResourceUpdateTopic,
		bus.InstanceHealthTopic,
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
				cp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigUploadRequestTopic, Data: message})
			case *mpi.ManagementPlaneRequest_ConfigApplyRequest:
				slog.Info("Config Apply Request")
				cp.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyRequestTopic, Data: message})
			default:
				slog.DebugContext(newCtx, "Management plane request not implemented yet")
			}
		}
	}
}
