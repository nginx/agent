// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"log/slog"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Instance struct {
	instances       []*instances.Instance
	messagePipe     bus.MessagePipeInterface
	instanceService service.InstanceServiceInterface
}

func NewInstance() *Instance {
	return &Instance{
		instances:       []*instances.Instance{},
		instanceService: service.NewInstanceService(),
	}
}

func (i *Instance) Init(messagePipe bus.MessagePipeInterface) {
	i.messagePipe = messagePipe
}

func (*Instance) Close() {}

func (*Instance) Info() *bus.Info {
	return &bus.Info{
		Name: "instance",
	}
}

func (i *Instance) Process(msg *bus.Message) {
	if msg.Topic == bus.OsProcessesTopic {
		newProcesses, ok := msg.Data.([]*model.Process)
		if !ok {
			slog.Error("unable to cast message payload to model.Process", "payload", msg.Data)
			return
		}

		instanceList := i.instanceService.GetInstances(newProcesses)
		if len(instanceList) > 0 {
			i.instances = instanceList
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
