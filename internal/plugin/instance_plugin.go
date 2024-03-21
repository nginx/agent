// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Instance struct {
	instances       []*instances.Instance
	messagePipe     bus.MessagePipeInterface
	instanceService service.InstanceServiceInterface
	instancesMutex  *sync.Mutex
}

func NewInstance() *Instance {
	return &Instance{
		instances:       []*instances.Instance{},
		instanceService: service.NewInstanceService(),
		instancesMutex:  &sync.Mutex{},
	}
}

func (i *Instance) Init(_ context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.Debug("Starting instance plugin")
	i.messagePipe = messagePipe

	return nil
}

func (*Instance) Close(_ context.Context) error {
	slog.Debug("Closing instance plugin")

	return nil
}

func (*Instance) Info() *bus.Info {
	return &bus.Info{
		Name: "instance",
	}
}

func (i *Instance) Process(_ context.Context, msg *bus.Message) {
	if msg.Topic == bus.OsProcessesTopic {
		newProcesses, ok := msg.Data.([]*model.Process)
		if !ok {
			slog.Error("unable to cast message payload to model.Process", "payload", msg.Data)
			return
		}

		instanceList := i.instanceService.GetInstances(newProcesses)
		if len(instanceList) > 0 {
			i.instancesMutex.Lock()
			i.instances = instanceList
			i.instancesMutex.Unlock()
			i.messagePipe.Process(&bus.Message{Topic: bus.InstancesTopic, Data: instanceList})
		} else {
			slog.Info("No instanceList found")
		}
	}
}

func (*Instance) Subscriptions() []string {
	return []string{
		bus.OsProcessesTopic,
		bus.InstanceConfigUpdateRequestTopic,
	}
}

func (i *Instance) getInstances() []*instances.Instance {
	i.instancesMutex.Lock()
	defer i.instancesMutex.Unlock()

	return i.instances
}
