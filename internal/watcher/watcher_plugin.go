// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
)

type (
	Watcher struct {
		messagePipe            bus.MessagePipeInterface
		agentConfig            *config.Config
		instanceWatcherService *InstanceWatcherService
		instanceUpdatesChannel chan InstanceUpdatesMessage
		cancel                 context.CancelFunc
	}
)

func NewWatcher(agentConfig *config.Config) *Watcher {
	return &Watcher{
		agentConfig:            agentConfig,
		instanceWatcherService: NewInstanceWatcherService(agentConfig),
		instanceUpdatesChannel: make(chan InstanceUpdatesMessage),
	}
}

// nolint: unparam
// error is always nil
func (w *Watcher) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting watcher plugin")
	w.messagePipe = messagePipe

	watcherContext, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	go w.instanceWatcherService.Watch(watcherContext, w.instanceUpdatesChannel)
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

func (*Watcher) Process(_ context.Context, _ *bus.Message) {}

func (*Watcher) Subscriptions() []string {
	return []string{}
}

func (w *Watcher) monitorWatchers(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-w.instanceUpdatesChannel:
			newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, message.correlationID)

			if len(message.instanceUpdates.newInstances) > 0 {
				slog.DebugContext(newCtx, "New instances found", "instances", message.instanceUpdates.newInstances)
				w.messagePipe.Process(
					newCtx,
					&bus.Message{Topic: bus.NewInstancesTopic, Data: message.instanceUpdates.newInstances},
				)
			}
			if len(message.instanceUpdates.deletedInstances) > 0 {
				slog.DebugContext(newCtx, "Instances deleted", "instances", message.instanceUpdates.deletedInstances)
				w.messagePipe.Process(
					newCtx,
					&bus.Message{Topic: bus.DeletedInstancesTopic, Data: message.instanceUpdates.deletedInstances},
				)
			}
		}
	}
}
