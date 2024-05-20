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
)

// nolint
type (
	Watcher struct {
		messagePipe            bus.MessagePipeInterface
		agentConfig            *config.Config
		instanceWatcherService *InstanceWatcherService
		instanceUpdatesChannel chan InstanceUpdates
		healthWatcherService   *HealthWatcherService
		cancel                 context.CancelFunc
	}
)

func NewWatcher(agentConfig *config.Config) *Watcher {
	return &Watcher{
		agentConfig:            agentConfig,
		instanceWatcherService: NewInstanceWatcherService(agentConfig),
		instanceUpdatesChannel: make(chan InstanceUpdates),
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
	go w.monitorInstanceUpdates(watcherContext)

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

func (w *Watcher) monitorInstanceUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case instanceUpdates := <-w.instanceUpdatesChannel:
			if len(instanceUpdates.newInstances) > 0 {
				slog.DebugContext(ctx, "New instances found", "instances", instanceUpdates.newInstances)
				w.messagePipe.Process(
					ctx,
					&bus.Message{Topic: bus.NewInstancesTopic, Data: instanceUpdates.newInstances},
				)
			}
			if len(instanceUpdates.deletedInstances) > 0 {
				slog.DebugContext(ctx, "Instances deleted", "instances", instanceUpdates.deletedInstances)
				w.messagePipe.Process(
					ctx,
					&bus.Message{Topic: bus.DeletedInstancesTopic, Data: instanceUpdates.deletedInstances},
				)
			}
		}
	}
}
