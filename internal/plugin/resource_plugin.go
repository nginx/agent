// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Resource struct {
	messagePipe     bus.MessagePipeInterface
	resourceService service.ResourceServiceInterface
	instanceService service.InstanceServiceInterface
	resource        *v1.Resource
	resourceMutex   sync.Mutex
}

func NewResource() *Resource {
	return &Resource{
		resourceMutex:   sync.Mutex{},
		resourceService: service.NewResourceService(),
		resource: &v1.Resource{
			Instances: []*v1.Instance{},
		},
	}
}

// nolint: unparam
// error is always nil
func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe

	r.resourceMutex.Lock()
	r.resource = r.resourceService.GetResource(ctx)
	r.resourceMutex.Unlock()

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
	case bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
			r.resourceMutex.Lock()
			r.resource.Instances = newInstances
			r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceTopic, Data: r.resource})
			r.resourceMutex.Unlock()
		}
	case bus.OsProcessesTopic:
		newProcesses, ok := msg.Data.([]*model.Process)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to model.Process", "payload", msg.Data)

			return
		}

		instanceList := r.instanceService.GetInstances(ctx, newProcesses)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.InstancesTopic, Data: instanceList})
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*Resource) Subscriptions() []string {
	return []string{
		bus.OsProcessesTopic,
		bus.InstancesTopic,
		// this was in instances plugin, will double check if needed
		bus.InstanceConfigUpdateRequestTopic,
	}
}
