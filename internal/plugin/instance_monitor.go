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
	"github.com/nginx/agent/v3/internal/datasource"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
	"github.com/nginx/agent/v3/internal/model/os"
)

type InstanceMonitor struct {
	instances       []*instances.Instance
	messagePipe     bus.MessagePipeInterface
	nginxDatasource datasource.Datasource
}

type InstanceMonitorParameters struct {
	nginxDatasource datasource.Datasource
}

func NewInstanceMonitor(instanceMonitorParameters *InstanceMonitorParameters) *InstanceMonitor {
	if instanceMonitorParameters.nginxDatasource == nil {
		instanceMonitorParameters.nginxDatasource = nginx.New(nginx.NginxParameters{})
	}
	return &InstanceMonitor{
		instances:       []*instances.Instance{},
		nginxDatasource: instanceMonitorParameters.nginxDatasource,
	}
}

func (im *InstanceMonitor) Init(messagePipe bus.MessagePipeInterface) error {
	im.messagePipe = messagePipe
	return nil
}

func (im *InstanceMonitor) Close() error { return nil }

func (im *InstanceMonitor) Info() *bus.Info {
	return &bus.Info{
		Name: "instance-monitor",
	}
}

func (im *InstanceMonitor) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.OS_PROCESSES_TOPIC:
		newProcesses := msg.Data.([]*os.Process)

		instances, err := im.nginxDatasource.GetInstances(newProcesses)
		if err != nil {
			slog.Warn("Unable to find NGINX instances", "error", err)
		} else {
			im.instances = instances
			im.messagePipe.Process(&bus.Message{Topic: bus.INSTANCES_TOPIC, Data: instances})
		}
	}
}

func (im *InstanceMonitor) Subscriptions() []string {
	return []string{
		bus.OS_PROCESSES_TOPIC,
	}
}
