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
	"github.com/nginx/agent/v3/internal/service"
)

type ResourceMonitor struct {
	messagePipe     bus.MessagePipeInterface
	resourceService service.ResourceServiceInterface
	resource        *v1.Resource
	resourceMutex   sync.Mutex
}

func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		resourceMutex:   sync.Mutex{},
		resourceService: service.NewResourceService(),
		resource:        &v1.Resource{},
	}
}

// nolint: unparam
// error is always nil
func (rm *ResourceMonitor) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource monitor plugin")

	rm.messagePipe = messagePipe

	rm.resourceMutex.Lock()
	rm.resource = rm.resourceService.GetResource(ctx)
	rm.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceTopic, Data: rm.resource})
	rm.resourceMutex.Unlock()

	return nil
}

func (*ResourceMonitor) Close(ctx context.Context) error {
	slog.DebugContext(ctx, "Closing resource monitor plugin")
	return nil
}

func (*ResourceMonitor) Info() *bus.Info {
	return &bus.Info{
		Name: "resource-monitor",
	}
}

func (rm *ResourceMonitor) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
			rm.resourceMutex.Lock()
			rm.resource.Instances = newInstances
			rm.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceTopic, Data: rm.resource})
			rm.resourceMutex.Unlock()
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*ResourceMonitor) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
	}
}
