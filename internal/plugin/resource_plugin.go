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
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
)

type Resource struct {
	messagePipe         bus.MessagePipeInterface
	resourceService     service.ResourceServiceInterface
	instanceService     service.InstanceServiceInterface
	resource            *v1.Resource
	resourceMutex       sync.Mutex
	nginxConfigContexts map[string]*model.NginxConfigContext
}

func NewResource(agentConfig *config.Config) *Resource {
	return &Resource{
		resourceMutex:   sync.Mutex{},
		resourceService: service.NewResourceService(),
		instanceService: service.NewInstanceService(agentConfig),
		resource: &v1.Resource{
			Instances: []*v1.Instance{},
		},
		nginxConfigContexts: make(map[string]*model.NginxConfigContext),
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
	case bus.OsProcessesTopic:
		newProcesses, ok := msg.Data.(map[int32]*model.Process)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to model.Process", "payload", msg.Data)

			return
		}

		instanceList := r.instanceService.GetInstances(ctx, newProcesses)
		r.resourceMutex.Lock()
		r.resource.Instances = instanceList
		r.updateNginxConfigContexts()
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceTopic, Data: r.resource})
		r.resourceMutex.Unlock()
	case bus.InstanceConfigContextTopic:
		nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to model.NginxConfigContext",
				"payload", msg.Data)
		}

		r.resourceMutex.Lock()
		instances := r.resource.GetInstances()
		for _, instance := range instances {
			if instance.GetInstanceMeta().GetInstanceId() == nginxConfigContext.InstanceID {
				r.updateInstance(nginxConfigContext, instance)
				r.nginxConfigContexts[instance.GetInstanceMeta().GetInstanceId()] = nginxConfigContext
			}
		}
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceTopic, Data: r.resource})
		r.resourceMutex.Unlock()
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (r *Resource) updateNginxConfigContexts() {
	for _, instance := range r.resource.GetInstances() {
		if val, configOk := r.nginxConfigContexts[instance.GetInstanceMeta().GetInstanceId()]; configOk {
			r.updateInstance(val, instance)
		}
	}
}

func (r *Resource) updateInstance(nginxConfigContext *model.NginxConfigContext, instance *v1.Instance) {
	if instance.GetInstanceRuntime().GetNginxRuntimeInfo() != nil {
		instanceRuntime := instance.GetInstanceRuntime().GetNginxRuntimeInfo()
		instanceRuntime.AccessLogs = convertAccessLogs(nginxConfigContext.AccessLogs)
		instanceRuntime.ErrorLogs = convertErrorLogs(nginxConfigContext.ErrorLogs)
		instanceRuntime.StubStatus = nginxConfigContext.StubStatus
	} else {
		instanceRuntime := instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo()
		instanceRuntime.AccessLogs = convertAccessLogs(nginxConfigContext.AccessLogs)
		instanceRuntime.ErrorLogs = convertErrorLogs(nginxConfigContext.ErrorLogs)
		instanceRuntime.StubStatus = nginxConfigContext.StubStatus
		instanceRuntime.PlusApi = nginxConfigContext.PlusAPI
	}
}

func convertAccessLogs(accessLogs []*model.AccessLog) (logs []string) {
	for _, log := range accessLogs {
		logs = append(logs, log.Name)
	}

	return logs
}

func convertErrorLogs(errorLogs []*model.ErrorLog) (logs []string) {
	for _, log := range errorLogs {
		logs = append(logs, log.Name)
	}

	return logs
}

func (*Resource) Subscriptions() []string {
	return []string{
		bus.OsProcessesTopic,
		bus.InstanceConfigContextTopic,
	}
}
