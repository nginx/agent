/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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

type InstanceParameters struct {
	instanceService service.InstanceServiceInterface
}

func NewInstance(instanceMonitorParameters *InstanceParameters) *Instance {
	if instanceMonitorParameters.instanceService == nil {
		instanceMonitorParameters.instanceService = service.NewInstanceService()
	}
	return &Instance{
		instances:       []*instances.Instance{},
		instanceService: instanceMonitorParameters.instanceService,
	}
}

func (i *Instance) Init(messagePipe bus.MessagePipeInterface) {
	i.messagePipe = messagePipe
}

func (i *Instance) Close() {}

func (i *Instance) Info() *bus.Info {
	return &bus.Info{
		Name: "instance",
	}
}

func (i *Instance) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.OS_PROCESSES_TOPIC:
		newProcesses := msg.Data.([]*model.Process)

		instances := i.instanceService.GetInstances(newProcesses)
		if len(instances) > 0 {
			i.instances = instances
			i.messagePipe.Process(&bus.Message{Topic: bus.INSTANCES_TOPIC, Data: instances})
		} else {
			slog.Info("No instances found")
		}
	}
}

func (i *Instance) Subscriptions() []string {
	return []string{
		bus.OS_PROCESSES_TOPIC,
		bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC,
	}
}
