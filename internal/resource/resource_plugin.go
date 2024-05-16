// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
)

type Resource struct {
	messagePipe     bus.MessagePipeInterface
	resourceService resourceServiceInterface
}

func NewResource() *Resource {
	return &Resource{
		resourceService: NewResourceService(),
	}
}

func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe

	return nil
}

func (*Resource) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing resource plugin")
	return nil
}

func (*Resource) Info() *bus.Info {
	return &bus.Info{
		Name: "resource",
	}
}

func (r *Resource) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.NewInstances:
		updatedResource, err := r.resourceService.AddInstance(msg)
		if err != nil {
			slog.ErrorContext(ctx, "Error adding new instance", "error", err)
		}

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: updatedResource})

		return
	case bus.UpdatedInstances:
		updatedResource, err := r.resourceService.UpdateInstance(msg)
		if err != nil {
			slog.ErrorContext(ctx, "Error updating instances", "error", err)
		}

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: updatedResource})

		return

	case bus.DeletedInstances:
		updatedResource, err := r.resourceService.DeleteInstance(msg)
		if err != nil {
			slog.ErrorContext(ctx, "Error deleting instances", "error", err)
		}

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: updatedResource})

		return
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*Resource) Subscriptions() []string {
	return []string{
		bus.NewInstances,
		bus.UpdatedInstances,
		bus.DeletedInstances,
	}
}
