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

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	response "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/bus"
)

// The resource plugin listens for a writeConfigSuccessfulTopic from the file plugin after the config apply
// files have been written. The resource plugin then, validates the config, reloads the instance and monitors the logs.
// This is done in the resource plugin to make the file plugin usable for every type of instance.

type Resource struct {
	messagePipe     bus.MessagePipeInterface
	resourceService resourceServiceInterface
	agentConfig     *config.Config
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

var _ bus.Plugin = (*Resource)(nil)

func NewResource(agentConfig *config.Config) *Resource {
	return &Resource{
		agentConfig: agentConfig,
	}
}

func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe
	r.resourceService = NewResourceService(ctx, r.agentConfig)

	return nil
}

func (*Resource) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing resource plugin")
	return nil
}

func (*Resource) Info() *bus.Info {
	return &bus.Info{
		Name: "resource",
	}
}

// cyclomatic complexity 11 max is 10
// nolint: revive, cyclop
func (r *Resource) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.AddInstancesTopic:
		slog.DebugContext(ctx, "Resource plugin received add instances message")
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}

		resource := r.resourceService.AddInstances(instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return
	case bus.UpdatedInstancesTopic:
		slog.DebugContext(ctx, "Resource plugin received update instances message")
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}
		resource := r.resourceService.UpdateInstances(ctx, instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return

	case bus.DeletedInstancesTopic:
		slog.DebugContext(ctx, "Resource plugin received delete instances message")
		instanceList, ok := msg.Data.([]*mpi.Instance)
		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

			return
		}
		resource := r.resourceService.DeleteInstances(ctx, instanceList)

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})

		return
	case bus.WriteConfigSuccessfulTopic:
		r.handleWriteConfigSuccessful(ctx, msg)
	case bus.RollbackWriteTopic:
		r.handleRollbackWrite(ctx, msg)
	case bus.APIActionRequestTopic:
		r.handleAPIActionRequest(ctx, msg)
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (*Resource) Subscriptions() []string {
	return []string{
		bus.AddInstancesTopic,
		bus.UpdatedInstancesTopic,
		bus.DeletedInstancesTopic,
		bus.WriteConfigSuccessfulTopic,
		bus.RollbackWriteTopic,
		bus.APIActionRequestTopic,
	}
}

func (r *Resource) handleAPIActionRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received api action request message")
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
		r.handleNginxPlusActionRequest(ctx, request.ActionRequest.GetNginxPlusAction(), instanceID)
	default:
		slog.DebugContext(ctx, "API action request not implemented yet")
	}
}

func (r *Resource) handleNginxPlusActionRequest(ctx context.Context, action *mpi.NGINXPlusAction, instanceID string) {
	correlationID := logger.CorrelationID(ctx)
	instance := r.resourceService.Instance(instanceID)
	apiAction := APIAction{
		ResourceService: r.resourceService,
	}
	if instance == nil {
		slog.ErrorContext(ctx, "Unable to find instance with ID", "id", instanceID)
		resp := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, fmt.Sprintf("failed to preform API "+
				"action, could not find instance with ID: %s", instanceID))

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})

		return
	}

	if instance.GetInstanceMeta().GetInstanceType() != mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		slog.ErrorContext(ctx, "Failed to preform API action", "error", errors.New("instance is not NGINX Plus"))
		resp := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"", instanceID, "failed to preform API action, instance is not NGINX Plus")

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})

		return
	}

	switch action.GetAction().(type) {
	case *mpi.NGINXPlusAction_UpdateHttpUpstreamServers:
		slog.DebugContext(ctx, "Updating http upstream servers", "request", action.GetUpdateHttpUpstreamServers())
		resp := apiAction.HandleUpdateHTTPUpstreamsRequest(ctx, action, instance)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetHttpUpstreamServers:
		slog.DebugContext(ctx, "Getting http upstream servers", "request", action.GetGetHttpUpstreamServers())
		resp := apiAction.HandleGetHTTPUpstreamsServersRequest(ctx, action, instance)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_UpdateStreamServers:
		slog.DebugContext(ctx, "Updating stream servers", "request", action.GetUpdateStreamServers())
		resp := apiAction.HandleUpdateStreamServersRequest(ctx, action, instance)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetStreamUpstreams:
		slog.DebugContext(ctx, "Getting stream upstreams", "request", action.GetGetStreamUpstreams())
		resp := apiAction.HandleGetStreamUpstreamsRequest(ctx, instance)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	case *mpi.NGINXPlusAction_GetUpstreams:
		slog.DebugContext(ctx, "Getting upstreams", "request", action.GetGetUpstreams())
		resp := apiAction.HandleGetUpstreamsRequest(ctx, instance)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})
	default:
		slog.DebugContext(ctx, "NGINX Plus action not implemented yet")
	}
}

func (r *Resource) handleWriteConfigSuccessful(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received write config successful message")
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	configContext, err := r.resourceService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		data.Error = err
		slog.ErrorContext(ctx, "errors found during config apply, "+
			"sending error status, rolling back config", "err", err)
		dpResponse := response.CreateDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config", data.InstanceID, err.Error())
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyFailedTopic, Data: data})

		return
	}

	dpResponse := response.CreateDataPlaneResponse(data.CorrelationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Config apply successful", data.InstanceID, "")

	successMessage := &model.ConfigApplySuccess{
		ConfigContext:     configContext,
		DataPlaneResponse: dpResponse,
	}

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplySuccessfulTopic, Data: successMessage})
}

func (r *Resource) handleRollbackWrite(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received rollback write message")
	data, ok := msg.Data.(*model.ConfigApplyMessage)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.ConfigApplyMessage", "payload", msg.Data)

		return
	}
	_, err := r.resourceService.ApplyConfig(ctx, data.InstanceID)
	if err != nil {
		slog.ErrorContext(ctx, "errors found during rollback, sending failure status", "err", err)

		rollbackResponse := response.CreateDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR, "Rollback failed", data.InstanceID, err.Error())

		applyResponse := response.CreateDataPlaneResponse(data.CorrelationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE, "Config apply failed, rollback failed",
			data.InstanceID, data.Error.Error())

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: applyResponse})

		return
	}

	applyResponse := response.CreateDataPlaneResponse(data.CorrelationID,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		"Config apply failed, rollback successful", data.InstanceID, data.Error.Error())

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ConfigApplyCompleteTopic, Data: applyResponse})
}
