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
	"github.com/nginx/agent/v3/internal/file"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/bus"
)

// The resource plugin listens for a writeConfigSuccessfulTopic from the file plugin after the config apply
// files have been written. The resource plugin then, validates the config, reloads the instance and monitors the logs.
// This is done in the resource plugin to make the file plugin usable for every type of instance.

type Resource struct {
	agentConfigMutex   *sync.Mutex
	manifestLock       *sync.RWMutex
	messagePipe        bus.MessagePipeInterface
	resourceService    resourceServiceInterface
	agentConfig        *config.Config
	conn               grpc.GrpcConnectionInterface
	fileManagerService *file.FileManagerService
	serverType         model.ServerType
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

func NewResource(agentConfig *config.Config, grpcConnection grpc.GrpcConnectionInterface,
	serverType model.ServerType, manifestLock *sync.RWMutex,
) *Resource {
	return &Resource{
		agentConfig:      agentConfig,
		conn:             grpcConnection,
		serverType:       serverType,
		manifestLock:     manifestLock,
		agentConfigMutex: &sync.Mutex{},
	}
}

func (r *Resource) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, r.serverType.String()),
	)
	slog.DebugContext(ctx, "Starting resource plugin")

	r.messagePipe = messagePipe
	r.resourceService = NewResourceService(ctx, r.agentConfig)
	r.fileManagerService = file.NewFileManagerService(r.conn.FileServiceClient(), r.agentConfig, r.manifestLock)

	return nil
}

func (r *Resource) Close(ctx context.Context) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, r.serverType.String()),
	)
	slog.InfoContext(ctx, "Closing resource plugin")

	return r.conn.Close(ctx)
}

func (r *Resource) Info() *bus.Info {
	name := "resource"
	if r.serverType.String() == model.Auxiliary.String() {
		name = "auxiliary-resource"
	}

	return &bus.Info{
		Name: name,
	}
}

//nolint:revive,cyclop // cyclomatic complexity 17 max is 10 switch statement cant be broken up
func (r *Resource) Process(ctx context.Context, msg *bus.Message) {
	ctxWithMetadata := r.agentConfig.NewContextWithLabels(ctx)

	if logger.ServerType(ctx) == "" {
		ctxWithMetadata = context.WithValue(
			ctxWithMetadata,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, r.serverType.String()),
		)
	}

	switch msg.Topic {
	// Remove and move logic to instance watcher or watcher plugin
	// This is to solve the issue of needing the resource plugin enabled for this logice
	//  but not having a GRPC connection. This change would make the rescource plugin behave more like the
	// file plugin and simplify the instance logic. Instead of different lists for add update delete it should be one upddated
	// list of instances that is sent
	//case bus.AddInstancesTopic:
	//	r.handleAddedInstances(ctx, msg)
	//case bus.UpdatedInstancesTopic:
	//	r.handleUpdatedInstances(ctx, msg)
	//case bus.DeletedInstancesTopic:
	//	r.handleDeletedInstances(ctx, msg)
	case bus.APIActionRequestTopic:
		r.handleAPIActionRequest(ctx, msg)
	case bus.ConnectionResetTopic:
		if logger.ServerType(ctxWithMetadata) == r.serverType.String() {
			r.handleConnectionReset(ctxWithMetadata, msg)
		}
	case bus.ConnectionCreatedTopic:
		if logger.ServerType(ctxWithMetadata) == r.serverType.String() {
			slog.DebugContext(ctxWithMetadata, "Resource plugin received connection created message")
			r.fileManagerService.SetIsConnected(true)
		}
	case bus.NginxConfigUpdateTopic:
		if logger.ServerType(ctxWithMetadata) == r.serverType.String() {
			r.handleNginxConfigUpdate(ctxWithMetadata, msg)
		}
	case bus.ConfigUploadRequestTopic:
		if logger.ServerType(ctxWithMetadata) == r.serverType.String() {
			r.handleConfigUploadRequest(ctxWithMetadata, msg)
		}
	case bus.ConfigApplyRequestTopic:
		if logger.ServerType(ctxWithMetadata) == r.serverType.String() {
			r.handleConfigApplyRequest(ctxWithMetadata, msg)
		}
	default:
		slog.DebugContext(ctx, "Resource plugin received unknown topic", "topic", msg.Topic)
	}
}

func (r *Resource) Subscriptions() []string {
	subscriptions := []string{
		bus.AddInstancesTopic,
		bus.UpdatedInstancesTopic,
		bus.DeletedInstancesTopic,
		bus.APIActionRequestTopic,
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
	}

	if r.serverType == model.Command {
		subscriptions = append(subscriptions, bus.ConfigApplyRequestTopic)
	}

	return subscriptions
}

func (r *Resource) Reconfigure(ctx context.Context, agentConfig *config.Config) error {
	slog.DebugContext(ctx, "Resource plugin is reconfiguring to update agent configuration")

	r.agentConfigMutex.Lock()
	defer r.agentConfigMutex.Unlock()

	r.agentConfig = agentConfig

	return nil
}

func (r *Resource) enableWatchers(ctx context.Context, configContext *model.NginxConfigContext,
	instanceID string,
) {
	enableWatcher := &model.EnableWatchers{
		ConfigContext: configContext,
		InstanceID:    instanceID,
	}

	r.messagePipe.Process(ctx, &bus.Message{
		Data:  enableWatcher,
		Topic: bus.EnableWatchersTopic,
	})
}

func (r *Resource) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received config upload request message")
	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(
			ctx,
			"Unable to cast message payload to *mpi.ManagementPlaneRequest",
			"payload", msg.Data,
		)

		return
	}

	configUploadRequest := managementPlaneRequest.GetConfigUploadRequest()

	correlationID := logger.CorrelationID(ctx)

	updatingFilesError := r.fileManagerService.ConfigUpload(ctx, configUploadRequest)

	dataPlaneResponse := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Successfully updated all files",
		},
	}

	if updatingFilesError != nil {
		dataPlaneResponse.CommandResponse.Status = mpi.CommandResponse_COMMAND_STATUS_FAILURE
		dataPlaneResponse.CommandResponse.Message = "Failed to update all files"
		dataPlaneResponse.CommandResponse.Error = updatingFilesError.Error()
	}

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dataPlaneResponse})
}

func (r *Resource) handleConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received connection reset message")
	if newConnection, ok := msg.Data.(grpc.GrpcConnectionInterface); ok {
		var reconnect bool
		err := r.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Resource plugin: unable to close connection", "error", err)
		}
		r.conn = newConnection

		reconnect = r.fileManagerService.IsConnected()
		r.fileManagerService.ResetClient(ctx, r.conn.FileServiceClient())
		r.fileManagerService.SetIsConnected(reconnect)

		slog.DebugContext(ctx, "File manager service client reset successfully")
	}
}

func (r *Resource) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received nginx config update message")
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	r.fileManagerService.ConfigUpdate(ctx, nginxConfigContext)
}

func (r *Resource) handleAddedInstances(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received add instances message")
	instanceList, ok := msg.Data.([]*mpi.Instance)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

		return
	}

	resource := r.resourceService.AddInstances(instanceList)

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})
}

func (r *Resource) handleUpdatedInstances(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received update instances message")
	instanceList, ok := msg.Data.([]*mpi.Instance)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

		return
	}
	resource := r.resourceService.UpdateInstances(ctx, instanceList)

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})
}

func (r *Resource) handleDeletedInstances(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received delete instances message")
	instanceList, ok := msg.Data.([]*mpi.Instance)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to []*mpi.Instance", "payload", msg.Data)

		return
	}
	resource := r.resourceService.DeleteInstances(ctx, instanceList)

	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: resource})
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

func (r *Resource) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "Resource plugin received config apply request message")

	var dataPlaneResponse *mpi.DataPlaneResponse
	correlationID := logger.CorrelationID(ctx)

	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest",
			"payload", msg.Data)

		return
	}

	request, requestOk := managementPlaneRequest.GetRequest().(*mpi.ManagementPlaneRequest_ConfigApplyRequest)
	if !requestOk {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest_ConfigApplyRequest",
			"payload", msg.Data)

		return
	}

	configApplyRequest := request.ConfigApplyRequest
	instanceID := configApplyRequest.GetOverview().GetConfigVersion().GetInstanceId()

	writeStatus, err := r.fileManagerService.ConfigApply(ctx, configApplyRequest)

	switch writeStatus {
	case model.NoChange:
		slog.DebugContext(ctx, "No changes required for config apply request")
		dataPlaneResponse = response.CreateDataPlaneResponse(correlationID,
			mpi.CommandResponse_COMMAND_STATUS_OK,
			"Config apply successful, no files to change",
			instanceID,
			"",
		)
		r.completeConfigApply(ctx, dataPlaneResponse)
	case model.Error:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes",
			"instance_id", instanceID,
			"error", err,
		)
		dataPlaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed",
			instanceID,
			err.Error(),
		)

		r.completeConfigApply(ctx, dataPlaneResponse)
	case model.RollbackRequired:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes, rolling back",
			"instance_id", instanceID,
			"error", err,
		)

		dataPlaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config",
			instanceID,
			err.Error(),
		)
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dataPlaneResponse})

		rollbackErr := r.fileManagerService.Rollback(
			ctx,
			instanceID,
		)
		if rollbackErr != nil {
			rollbackResponse := response.CreateDataPlaneResponse(
				correlationID,
				mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				"Config apply failed, rollback failed",
				instanceID,
				rollbackErr.Error())

			r.completeConfigApply(ctx, rollbackResponse)

			return
		}

		dataPlaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed, rollback successful",
			instanceID,
			err.Error())

		r.completeConfigApply(ctx, dataPlaneResponse)
	case model.OK:
		slog.DebugContext(ctx, "Changes required for config apply request")
		r.applyConfig(ctx, correlationID, instanceID)
	}
}

func (r *Resource) completeConfigApply(ctx context.Context, dpResponse *mpi.DataPlaneResponse) {
	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})
	r.fileManagerService.ClearCache()
	r.enableWatchers(ctx, &model.NginxConfigContext{}, dpResponse.GetInstanceId())
}

func (r *Resource) applyConfig(ctx context.Context, correlationID, instanceID string) {
	slog.DebugContext(ctx, "Resource plugin received write config successful message")

	configContext, err := r.resourceService.ApplyConfig(ctx, instanceID)
	if err != nil {
		slog.ErrorContext(ctx, "errors found during config apply, "+
			"sending error status, rolling back config", "err", err)
		dpResponse := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Config apply failed, rolling back config", instanceID, err.Error())
		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})

		r.failedConfigApply(ctx, correlationID, instanceID, err)

		return
	}

	dpResponse := response.CreateDataPlaneResponse(correlationID, mpi.CommandResponse_COMMAND_STATUS_OK,
		"Config apply successful", instanceID, "")

	r.reloadSuccessful(ctx, configContext, dpResponse)
}

func (r *Resource) failedConfigApply(ctx context.Context, correlationID, instanceID string, applyErr error) {
	if instanceID == "" {
		r.fileManagerService.ClearCache()

		return
	}

	err := r.fileManagerService.Rollback(ctx, instanceID)
	if err != nil {
		rollbackResponse := response.CreateDataPlaneResponse(correlationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR,
			"Rollback failed", instanceID, err.Error())

		applyResponse := response.CreateDataPlaneResponse(correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			"Config apply failed, rollback failed", instanceID, applyErr.Error())

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})

		r.completeConfigApply(ctx, applyResponse)

		return
	}

	r.handleRollbackWrite(ctx, correlationID, instanceID, applyErr)
}

func (r *Resource) handleRollbackWrite(ctx context.Context, correlationID, instanceID string, applyErr error) {
	slog.DebugContext(ctx, "Resource plugin received rollback write message")

	_, err := r.resourceService.ApplyConfig(ctx, instanceID)
	if err != nil {
		slog.ErrorContext(ctx, "errors found during rollback, sending failure status", "err", err)

		rollbackResponse := response.CreateDataPlaneResponse(correlationID,
			mpi.CommandResponse_COMMAND_STATUS_ERROR, "Rollback failed", instanceID, err.Error())

		applyResponse := response.CreateDataPlaneResponse(correlationID,
			mpi.CommandResponse_COMMAND_STATUS_FAILURE, "Config apply failed, rollback failed",
			instanceID, applyErr.Error())

		r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})

		r.completeConfigApply(ctx, applyResponse)

		return
	}

	applyResponse := response.CreateDataPlaneResponse(correlationID,
		mpi.CommandResponse_COMMAND_STATUS_FAILURE,
		"Config apply failed, rollback successful", instanceID, applyErr.Error())

	r.completeConfigApply(ctx, applyResponse)
}

func (r *Resource) reloadSuccessful(ctx context.Context,
	configContext *model.NginxConfigContext, dpResponse *mpi.DataPlaneResponse,
) {
	r.fileManagerService.ClearCache()
	r.enableWatchers(ctx, configContext, dpResponse.GetInstanceId())

	if configContext.Files != nil {
		slog.DebugContext(ctx, "Changes made during config apply, update files on disk")
		updateError := r.fileManagerService.UpdateCurrentFilesOnDisk(
			ctx,
			files.ConvertToMapOfFiles(configContext.Files),
			true,
		)
		if updateError != nil {
			slog.ErrorContext(ctx, "Unable to update current files on disk", "error", updateError)
		}
	}
	r.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})
}
