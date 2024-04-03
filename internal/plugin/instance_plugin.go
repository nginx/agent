// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Instance struct {
	messagePipe     bus.MessagePipeInterface
	instanceService service.InstanceServiceInterface
}

func NewInstance() *Instance {
	return &Instance{
		instanceService: service.NewInstanceService(),
	}
}

func (i *Instance) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting instance plugin")
	i.messagePipe = messagePipe

	return nil
}

func (*Instance) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing instance plugin")

	return nil
}

func (*Instance) Info() *bus.Info {
	return &bus.Info{
		Name: "instance",
	}
}

func (i *Instance) Process(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Instance plugin process")
	if msg.Topic == bus.OsProcessesTopic {
		newProcesses, ok := msg.Data.([]*model.Process)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to model.Process", "payload", msg.Data)

			return
		}

		instanceList := i.instanceService.GetInstances(ctx, newProcesses)
		if len(instanceList) > 0 {
			i.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstancesTopic, Data: instanceList})
		} else {
			slog.InfoContext(ctx, "No instances found")
		}
	}
}

func (*Instance) Subscriptions() []string {
	return []string{
		bus.OsProcessesTopic,
		bus.InstanceConfigUpdateRequestTopic,
	}
}
