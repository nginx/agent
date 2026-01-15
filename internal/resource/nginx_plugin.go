// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	response "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/bus"
)

// The Nginx plugin listens for a writeConfigSuccessfulTopic from the file plugin after the config apply
// files have been written. The Nginx plugin then, validates the config, reloads the instance and monitors the logs.
// This is done in the Nginx plugin to make the file plugin usable for every type of instance.

type Nginx struct {
	messagePipe      bus.MessagePipeInterface
	nginxService     nginxServiceInterface
	agentConfig      *config.Config
	agentConfigMutex *sync.Mutex
}

type errResponse struct {
	Status string `json:"status"`
	Text   string `json:"test"`
	Code   string `json:"code"`
}

type plusAPIErr struct {
	Error     errResponse `json:"error"`
	RequestID string      `json:"request_id"`
	Href      string      `json:"href"`
}

var _ bus.Plugin = (*Nginx)(nil)

func NewNginx(agentConfig *config.Config) *Nginx {
	return &Nginx{
		agentConfig:      agentConfig,
		agentConfigMutex: &sync.Mutex{},
	}
}

func (n *Nginx) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting nginx plugin")

	n.messagePipe = messagePipe
	n.nginxService = NewNginxService(ctx, n.agentConfig)

	return nil
}

func (*Nginx) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing nginx plugin")
	return nil
}

func (*Nginx) Info() *bus.Info {
	return &bus.Info{
		Name: "nginx",
	}
}

// cyclomatic complexity 11 max is 10

func (n *Nginx) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ResourceUpdateTopic:
		resourceUpdate, ok := msg.Data.(*mpi.Resource)

		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.Resource", "payload",
				msg.Data)

			return
		}
		n.nginxService.UpdateResource(ctx, resourceUpdate)
		slog.DebugContext(ctx, "Nginx plugin received update resource message")

		return
	case bus.WriteConfigSuccessfulTopic:
		n.handleWriteConfigSuccessful(ctx, msg)
	case bus.RollbackWriteTopic:
		n.handleRollbackWrite(ctx, msg)
	case bus.APIActionRequestTopic:
		n.handleAPIActionRequest(ctx, msg)
	case bus.AgentConfigUpdateTopic:
		n.handleAgentConfigUpdate(ctx, msg)
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*Nginx) Subscriptions() []string {
	return []string{
		bus.ResourceUpdateTopic,
		bus.WriteConfigSuccessfulTopic,
		bus.RollbackWriteTopic,
		bus.APIActionRequestTopic,
		bus.AgentConfigUpdateTopic,
	}
}

func (n *Nginx) Reconfigure(ctx context.Context, agentConfig *config.Config) error {
	slog.DebugContext(ctx, "Nginx plugin is reconfiguring to update agent configuration")

	n.agentConfigMutex.Lock()
	defer n.agentConfigMutex.Unlock()

	n.agentConfig = agentConfig

	return nil
}

func (n *Nginx) handleAPIActionRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Nginx plugin received api action request message")
	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)

	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest", "payload",
			msg.Data)

		return
	}

	request, requestOk := managementPlaneRequest.GetRequest().(*mpi.ManagementPlaneRequest_ActionRequest)
	if !requestOk {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ActionRequest",
			"payload", msg.Data)
	}

	instanceID := request.ActionRequest.GetInstanceId()

	switch request.ActionRequest.GetAction().(type) {
	case *mpi.APIActionRequest_NginxPlusAction:
		n.handleNginxPlusActionRequest(ctx, request.ActionRequest.GetNginxPlusAction(), instanceID)
	default:
		slog.DebugContext(ctx, "API action request not implemented yet")
	}
}

func (n *Nginx) handleNginxPlusActionRequest(ctx context.Context, action *mpi.NGINXPlusAction, instanceID string) {
	correlationID := logger.CorrelationID(ctx)
	instance := n.nginxService.Instance(instanceID)
	apiAction := APIAction{
		NginxService: n.nginxService,
	}
	if instance == nil {
		slog.ErrorContext(ctx, "Unable to find instance with ID", "id", instanceID)
		resp := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, fmt.Sprintf("failed to preform API "+
				"action, could not find instance with ID: %s", instanceID))

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})

		return
	}

	if instance.GetInstanceMeta().GetInstanceType() != mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		slog.ErrorContext(ctx, "Failed to preform API action", "error", errors.New("instance is not NGINX Plus"))
		resp := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, "failed to preform API action, instance is not NGINX Plus")

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})

		return
	}

	switch action.GetAction().(type) {
	case *mpi.NGINXPlusAction_UpdateHttpUpstreamServers:
		slog.DebugContext(ctx, "Updating http upstream servers", "request", action.GetUpdateHttpUpstreamServers())
		resp := apiAction.HandleUpdateHTTPUpstreamsRequest(ctx, action, instance)
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetHttpUpstreamServers:
		slog.DebugContext(ctx, "Getting http upstream servers", "request", action.GetGetHttpUpstreamServers())
		resp := apiAction.HandleGetHTTPUpstreamsServersRequest(ctx, action, instance)
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_UpdateStreamServers:
		slog.DebugContext(ctx, "Updating stream servers", "request", action.GetUpdateStreamServers())
		resp := apiAction.HandleUpdateStreamServersRequest(ctx, action, instance)
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetStreamUpstreams:
		slog.DebugContext(ctx, "Getting stream upstreams", "request", action.GetGetStreamUpstreams())
		resp := apiAction.HandleGetStreamUpstreamsRequest(ctx, instance)
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetUpstreams:
		slog.DebugContext(ctx, "Getting upstreams", "request", action.GetGetUpstreams())
		resp := apiAction.HandleGetUpstreamsRequest(ctx, instance)
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	default:
		slog.DebugContext(ctx, "NGINX Plus action not implemented yet")
	}
}

func (n *Nginx) handleWriteConfigSuccessful(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Nginx plugin received write config successful message")
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	configContext, err := n.nginxService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		data.Error = err
		slog.ErrorContext(ctx, "errors found during config apply, "+
			"sending error status, rolling back config", "err", err)
		dpResponse := response.CreateDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config", data.InstanceID, err.Error())
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyFailedTopic, Data: data})

		return
	}

	dpResponse := response.CreateDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Config apply successful", data.InstanceID, "")

	successMessage := &model.ReloadSuccess{
		ConfigContext:     configContext,
		DataPlaneResponse: dpResponse,
	}

	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.ReloadSuccessfulTopic, Data: successMessage})
}

func (n *Nginx) handleRollbackWrite(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Nginx plugin received rollback write message")
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	_, err := n.nginxService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		slog.ErrorContext(ctx, "errors found during rollback, sending failure status", "err", err)

		rollbackResponse := response.CreateDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR, "Rollback failed", data.InstanceID, err.Error())

		applyResponse := response.CreateDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE, "Config apply failed, rollback failed",
			data.InstanceID, data.Error.Error())

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})
		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: applyResponse})

		return
	}

	applyResponse := response.CreateDataPlaneResponse(data.CorrelationID,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		"Config apply failed, rollback successful", data.InstanceID, data.Error.Error())

	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: applyResponse})
}

func (n *Nginx) handleAgentConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Nginx plugin received agent config update message")

	n.agentConfigMutex.Lock()
	defer n.agentConfigMutex.Unlock()

	agentConfig, ok := msg.Data.(*config.Config)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *config.Config", "payload", msg.Data)
		return
	}

	n.agentConfig = agentConfig
}
