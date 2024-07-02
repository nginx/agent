// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/internal/watcher/health"
	"github.com/nginx/agent/v3/internal/watcher/instance"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . instanceWatcherServiceInterface

// nolint
type (
	Watcher struct {
		messagePipe               bus.MessagePipeInterface
		agentConfig               *config.Config
		instanceWatcherService    instanceWatcherServiceInterface
		healthWatcherService      *health.HealthWatcherService
		instanceUpdatesChannel    chan instance.InstanceUpdatesMessage
		nginxConfigContextChannel chan instance.NginxConfigContextMessage
		instanceHealthChannel     chan health.InstanceHealthMessage
		cancel                    context.CancelFunc
	}

	instanceWatcherServiceInterface interface {
		Watch(
			ctx context.Context,
			instancesChannel chan<- instance.InstanceUpdatesMessage,
			nginxConfigContextChannel chan<- instance.NginxConfigContextMessage,
		)
		ReparseConfig(ctx context.Context, instance *mpi.Instance)
	}
)

var _ bus.Plugin = (*Watcher)(nil)

func NewWatcher(agentConfig *config.Config) *Watcher {
	return &Watcher{
		agentConfig:               agentConfig,
		instanceWatcherService:    instance.NewInstanceWatcherService(agentConfig),
		healthWatcherService:      health.NewHealthWatcherService(agentConfig),
		instanceUpdatesChannel:    make(chan instance.InstanceUpdatesMessage),
		nginxConfigContextChannel: make(chan instance.NginxConfigContextMessage),
		instanceHealthChannel:     make(chan health.InstanceHealthMessage),
	}
}

// nolint: unparam
// error is always nil
func (w *Watcher) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting watcher plugin")
	w.messagePipe = messagePipe

	watcherContext, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go w.instanceWatcherService.Watch(watcherContext, w.instanceUpdatesChannel, w.nginxConfigContextChannel)
	go w.healthWatcherService.Watch(watcherContext, w.instanceHealthChannel)
	go w.monitorWatchers(watcherContext)

	return nil
}

// nolint: unparam
// error is always nil
func (w *Watcher) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing watcher plugin")

	w.cancel()

	return nil
}

func (*Watcher) Info() *bus.Info {
	return &bus.Info{
		Name: "watcher",
	}
}

func (w *Watcher) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ConfigApplySuccessfulTopic:
		data, ok := msg.Data.(*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to Instance", "payload", msg.Data)
			return
		}
		w.instanceWatcherService.ReparseConfig(ctx, data)
	default:
		slog.DebugContext(ctx, "Watcher plugin unknown topic", "topic", msg.Topic)
	}
}

func (*Watcher) Subscriptions() []string {
	return []string{
		bus.ConfigApplySuccessfulTopic,
	}
}

func (w *Watcher) monitorWatchers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-w.instanceUpdatesChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			w.handleInstanceUpdates(newCtx, message)
		case message := <-w.nginxConfigContextChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			slog.DebugContext(
				newCtx,
				"Updated NGINX config context",
				"nginx_config_context", message.NginxConfigContext,
			)
			w.messagePipe.Process(
				newCtx,
				&bus.Message{Topic: bus.NginxConfigUpdateTopic, Data: message.NginxConfigContext},
			)
		case message := <-w.instanceHealthChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.CorrelationID)
			w.messagePipe.Process(newCtx, &bus.Message{
				Topic: bus.InstanceHealthTopic, Data: message.InstanceHealth,
			})
		}
	}
}

func (w *Watcher) handleInstanceUpdates(newCtx context.Context, message instance.InstanceUpdatesMessage) {
	if len(message.InstanceUpdates.NewInstances) > 0 {
		slog.DebugContext(newCtx, "New instances found", "instances", message.InstanceUpdates.NewInstances)
		w.healthWatcherService.AddHealthWatcher(message.InstanceUpdates.NewInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.AddInstancesTopic, Data: message.InstanceUpdates.NewInstances},
		)
	}
	if len(message.InstanceUpdates.UpdatedInstances) > 0 {
		slog.DebugContext(newCtx, "Instances updated", "instances", message.InstanceUpdates.UpdatedInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.UpdatedInstancesTopic, Data: message.InstanceUpdates.UpdatedInstances},
		)
	}
	if len(message.InstanceUpdates.DeletedInstances) > 0 {
		slog.DebugContext(newCtx, "Instances deleted", "instances", message.InstanceUpdates.DeletedInstances)
		w.healthWatcherService.DeleteHealthWatcher(message.InstanceUpdates.
			DeletedInstances)
		w.messagePipe.Process(
			newCtx,
			&bus.Message{Topic: bus.DeletedInstancesTopic, Data: message.InstanceUpdates.DeletedInstances},
		)
	}
}
