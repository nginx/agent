// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/internal/bus"
)

type Resource struct {
	messagePipe     bus.MessagePipeInterface
	resourceService resourceServiceInterface
	resource        *v1.Resource
}

func NewResource() *Resource {
	return &Resource{
		resourceService: NewResourceService(),
	}
}

func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe
	r.resource = r.resourceService.GetResource(ctx)

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
		instanceList, ok := msg.Data.([]*v1.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*v1.Instance", "payload", msg.Data)
		}

		r.resource = r.resourceService.AddInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: r.resource})

		return
	case bus.UpdatedInstances:
		instanceList, ok := msg.Data.([]*v1.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*v1.Instance", "payload", msg.Data)
		}
		r.resource = r.resourceService.UpdateInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: r.resource})

		return

	case bus.DeletedInstances:
		instanceList, ok := msg.Data.([]*v1.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*v1.Instance", "payload", msg.Data)
		}
		r.resource = r.resourceService.DeleteInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdate, Data: r.resource})

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
