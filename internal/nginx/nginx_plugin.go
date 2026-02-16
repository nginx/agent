// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	response "github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/file"
	grpc2 "github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// The NginxPlugin plugin listens for a writeConfigSuccessfulTopic from the file plugin after the config apply
// files have been written. The NginxPlugin plugin then, validates the config,
// reloads the instance and monitors the logs.
// This is done in the NginxPlugin plugin to make the file plugin usable for every type of instance.

type NginxPlugin struct {
	messagePipe        bus.MessagePipeInterface
	nginxService       nginxServiceInterface
	agentConfig        *config.Config
	agentConfigMutex   *sync.Mutex
	manifestLock       *sync.RWMutex
	conn               grpc2.GrpcConnectionInterface
	fileManagerService file.FileManagerServiceInterface
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

var _ bus.Plugin = (*NginxPlugin)(nil)

func NewNginx(agentConfig *config.Config, grpcConnection grpc2.GrpcConnectionInterface,
	serverType model.ServerType, manifestLock *sync.RWMutex,
) *NginxPlugin {
	return &NginxPlugin{
		agentConfig:      agentConfig,
		conn:             grpcConnection,
		serverType:       serverType,
		manifestLock:     manifestLock,
		agentConfigMutex: &sync.Mutex{},
	}
}

func (n *NginxPlugin) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, n.serverType.String()),
	)
	slog.DebugContext(ctx, "Starting nginx plugin")

	n.messagePipe = messagePipe
	n.nginxService = NewNginxService(ctx, n.agentConfig)
	n.fileManagerService = file.NewFileManagerService(n.conn.FileServiceClient(), n.agentConfig, n.manifestLock)

	return nil
}

func (n *NginxPlugin) Close(ctx context.Context) error {
	ctx = context.WithValue(
		ctx,
		logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, n.serverType.String()),
	)
	slog.InfoContext(ctx, "Closing nginx plugin")

	return n.conn.Close(ctx)
}

func (n *NginxPlugin) Info() *bus.Info {
	name := "nginx"
	if n.serverType.String() == model.Auxiliary.String() {
		name = "auxiliary-nginx"
	}

	return &bus.Info{
		Name: name,
	}
}

//nolint:revive,cyclop  // cyclomatic complexity 16 max is 12
func (n *NginxPlugin) Process(ctx context.Context, msg *bus.Message) {
	ctxWithMetadata := n.agentConfig.NewContextWithLabels(ctx)
	if logger.ServerType(ctx) == "" {
		ctxWithMetadata = context.WithValue(
			ctxWithMetadata,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, n.serverType.String()),
		)
	}

	switch msg.Topic {
	case bus.ResourceUpdateTopic:
		resourceUpdate, ok := msg.Data.(*mpi.Resource)

		if !ok {
			slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.Resource", "payload",
				msg.Data)

			return
		}
		n.nginxService.UpdateResource(ctx, resourceUpdate)
		slog.DebugContext(ctx, "NginxPlugin plugin received update resource message")

		return
	case bus.APIActionRequestTopic:
		n.handleAPIActionRequest(ctx, msg)
	case bus.ConnectionResetTopic:
		if logger.ServerType(ctxWithMetadata) == n.serverType.String() {
			n.handleConnectionReset(ctxWithMetadata, msg)
		}
	case bus.ConnectionCreatedTopic:
		if logger.ServerType(ctxWithMetadata) == n.serverType.String() {
			slog.DebugContext(ctxWithMetadata, "Resource plugin received connection created message")
			n.fileManagerService.SetIsConnected(true)
		}
	case bus.NginxConfigUpdateTopic:
		if logger.ServerType(ctxWithMetadata) == n.serverType.String() {
			n.handleNginxConfigUpdate(ctxWithMetadata, msg)
		}
	case bus.ConfigUploadRequestTopic:
		if logger.ServerType(ctxWithMetadata) == n.serverType.String() {
			n.handleConfigUploadRequest(ctxWithMetadata, msg)
		}
	case bus.ConfigApplyRequestTopic:
		if logger.ServerType(ctxWithMetadata) == n.serverType.String() {
			n.handleConfigApplyRequest(ctxWithMetadata, msg)
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (n *NginxPlugin) Subscriptions() []string {
	subscriptions := []string{
		bus.APIActionRequestTopic,
		bus.ConnectionResetTopic,
		bus.ConnectionCreatedTopic,
		bus.NginxConfigUpdateTopic,
		bus.ConfigUploadRequestTopic,
		bus.ResourceUpdateTopic,
	}

	if n.serverType == model.Command {
		subscriptions = append(subscriptions, bus.ConfigApplyRequestTopic)
	}

	return subscriptions
}

func (n *NginxPlugin) Reconfigure(ctx context.Context, agentConfig *config.Config) error {
	slog.DebugContext(ctx, "NginxPlugin plugin is reconfiguring to update agent configuration")

	n.agentConfigMutex.Lock()
	defer n.agentConfigMutex.Unlock()

	n.agentConfig = agentConfig

	return nil
}

func (n *NginxPlugin) enableWatchers(ctx context.Context, configContext *model.NginxConfigContext, instanceID string) {
	enableWatcher := &model.EnableWatchers{
		InstanceID:    instanceID,
		ConfigContext: configContext,
	}

	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.EnableWatchersTopic, Data: enableWatcher})
}

func (n *NginxPlugin) handleConfigUploadRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "NginxPlugin plugin received config upload request message")
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

	updatingFilesError := n.fileManagerService.ConfigUpload(ctx, configUploadRequest)

	dataplaneResponse := &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Successfully updated all files",
		},
		InstanceId:  configUploadRequest.GetOverview().GetConfigVersion().GetInstanceId(),
		RequestType: mpi.DataPlaneResponse_CONFIG_UPLOAD_REQUEST,
	}

	if updatingFilesError != nil {
		dataplaneResponse.CommandResponse.Status = mpi.CommandResponse_COMMAND_STATUS_FAILURE
		dataplaneResponse.CommandResponse.Message = "Failed to update all files"
		dataplaneResponse.CommandResponse.Error = updatingFilesError.Error()
	}

	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dataplaneResponse})
}

func (n *NginxPlugin) handleConnectionReset(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "NginxPlugin plugin received connection reset message")

	if newConnection, ok := msg.Data.(grpc2.GrpcConnectionInterface); ok {
		err := n.conn.Close(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "NginxPlugin plugin: unable to close connection", "error", err)
		}

		n.conn = newConnection

		reconnect := n.fileManagerService.IsConnected()
		n.fileManagerService.ResetClient(ctx, n.conn.FileServiceClient())
		n.fileManagerService.SetIsConnected(reconnect)

		slog.DebugContext(ctx, "NginxPlugin plugin connection reset successfully")
	}
}

func (n *NginxPlugin) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "NginxPlugin plugin received config update message")
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)

	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)

		return
	}

	n.fileManagerService.ConfigUpdate(ctx, nginxConfigContext)
}

func (n *NginxPlugin) handleConfigApplyRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "NginxPlugin plugin received config apply request message")

	var dataplaneResponse *mpi.DataPlaneResponse
	correlationID := logger.CorrelationID(ctx)

	managementPlaneRequest, ok := msg.Data.(*mpi.ManagementPlaneRequest)

	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *mpi.ManagementPlaneRequest", "payload", msg.Data)
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

	writeStatus, err := n.fileManagerService.ConfigApply(ctx, configApplyRequest)

	switch writeStatus {
	case model.NoChange:
		slog.DebugContext(ctx, "No changes required for config apply request")
		dataplaneResponse = response.CreateDataPlaneResponse(correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
				Message: "Config apply successful, no files to change",
				Error:   "",
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)
		n.completeConfigApply(ctx, dataplaneResponse)
	case model.Error:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes",
			"instance_id", instanceID,
			"error", err,
		)
		dataplaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config apply failed",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		n.completeConfigApply(ctx, dataplaneResponse)
	case model.RollbackRequired:
		slog.ErrorContext(
			ctx,
			"Failed to apply config changes, rolling back",
			"instance_id", instanceID,
			"error", err,
		)

		dataplaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
				Message: "Config apply failed, rolling back config",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dataplaneResponse})

		rollbackErr := n.fileManagerService.Rollback(ctx, instanceID)

		if rollbackErr != nil {
			applyErr := fmt.Errorf("config apply error: %w", err)
			rbErr := fmt.Errorf("rollback error: %w", rollbackErr)
			combinedErr := errors.Join(applyErr, rbErr)

			rollbackResponse := response.CreateDataPlaneResponse(
				correlationID,
				&mpi.CommandResponse{
					Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
					Message: "Config apply failed, rollback failed",
					Error:   combinedErr.Error(),
				},
				mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
				instanceID,
			)
			n.completeConfigApply(ctx, rollbackResponse)

			return
		}

		dataplaneResponse = response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config apply failed, rollback successful",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID)
		n.completeConfigApply(ctx, dataplaneResponse)
	case model.OK:
		slog.DebugContext(ctx, "Changes required for config apply request")
		n.applyConfig(ctx, correlationID, instanceID)
	}
}

func (n *NginxPlugin) applyConfig(ctx context.Context, correlationID, instanceID string) {
	configContext, err := n.nginxService.ApplyConfig(ctx, instanceID)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Errors found during config apply, sending error status and rolling back configuration updates",
			"error", err,
		)
		dpResponse := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
				Message: "Config apply failed, rolling back config",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})

		n.failedConfigApply(ctx, correlationID, instanceID, err)

		return
	}

	dpResponse := response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: "Config apply successful",
			Error:   "",
		},
		mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
		instanceID,
	)

	n.reloadSuccessful(ctx, configContext, dpResponse)
}

func (n *NginxPlugin) failedConfigApply(ctx context.Context, correlationID, instanceID string, applyErr error) {
	if instanceID == "" {
		n.fileManagerService.ClearCache()
		return
	}

	err := n.fileManagerService.Rollback(ctx, instanceID)
	if err != nil {
		configErr := fmt.Errorf("config apply error: %w", applyErr)
		rbErr := fmt.Errorf("rollback error: %w", err)
		combinedErr := errors.Join(configErr, rbErr)

		rollbackResponse := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
				Message: "Rollback failed",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		applyResponse := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config apply failed, rollback failed",
				Error:   combinedErr.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})

		n.completeConfigApply(ctx, applyResponse)

		return
	}

	n.handleRollbackWrite(ctx, correlationID, instanceID, applyErr)
}

func (n *NginxPlugin) completeConfigApply(ctx context.Context, dpResponse *mpi.DataPlaneResponse) {
	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})
	n.fileManagerService.ClearCache()
	n.enableWatchers(ctx, &model.NginxConfigContext{}, dpResponse.GetInstanceId())
}

func (n *NginxPlugin) handleAPIActionRequest(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "NginxPlugin plugin received api action request message")
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

func (n *NginxPlugin) handleNginxPlusActionRequest(ctx context.Context,
	action *mpi.NGINXPlusAction, instanceID string,
) {
	correlationID := logger.CorrelationID(ctx)
	instance := n.nginxService.Instance(instanceID)
	apiAction := APIAction{
		NginxService: n.nginxService,
	}
	if instance == nil {
		slog.ErrorContext(ctx, "Unable to find instance with ID", "id", instanceID)
		resp := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   "failed to preform API action, could not find instance with ID: " + instanceID,
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: resp})

		return
	}

	if instance.GetInstanceMeta().GetInstanceType() != mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
		slog.ErrorContext(ctx, "Failed to preform API action", "error", errors.New("instance is not NGINX Plus"))
		resp := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "",
				Error:   "failed to preform API action, instance is not NGINX Plus",
			},
			mpi.DataPlaneResponse_API_ACTION_REQUEST,
			instanceID,
		)

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

func (n *NginxPlugin) handleRollbackWrite(ctx context.Context, correlationID, instanceID string, applyErr error) {
	slog.DebugContext(ctx, "NginxPlugin plugin received rollback write message")
	_, err := n.nginxService.ApplyConfig(ctx, instanceID)
	if err != nil {
		slog.ErrorContext(ctx, "Errors found during rollback, sending failure status", "error", err)

		rollbackResponse := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_ERROR,
				Message: "Rollback failed",
				Error:   err.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		configErr := fmt.Errorf("config apply error: %w", applyErr)
		rbErr := fmt.Errorf("rollback error: %w", err)
		combinedErr := errors.Join(configErr, rbErr)

		applyResponse := response.CreateDataPlaneResponse(
			correlationID,
			&mpi.CommandResponse{
				Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
				Message: "Config apply failed, rollback failed",
				Error:   combinedErr.Error(),
			},
			mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
			instanceID,
		)

		n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: rollbackResponse})

		n.completeConfigApply(ctx, applyResponse)

		return
	}

	applyResponse := response.CreateDataPlaneResponse(
		correlationID,
		&mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_FAILURE,
			Message: "Config apply failed, rollback successful",
			Error:   applyErr.Error(),
		},
		mpi.DataPlaneResponse_CONFIG_APPLY_REQUEST,
		instanceID,
	)

	n.completeConfigApply(ctx, applyResponse)
}

func (n *NginxPlugin) reloadSuccessful(ctx context.Context,
	configContext *model.NginxConfigContext, dpResponse *mpi.DataPlaneResponse,
) {
	n.fileManagerService.ClearCache()
	n.enableWatchers(ctx, configContext, dpResponse.GetInstanceId())

	if configContext.Files != nil {
		slog.DebugContext(ctx, "Changes made during config apply, update files on disk")
		updateError := n.fileManagerService.UpdateCurrentFilesOnDisk(
			ctx,
			files.ConvertToMapOfFiles(configContext.Files),
			true,
		)
		if updateError != nil {
			slog.ErrorContext(ctx, "Unable to update current files on disk", "error", updateError)
		}
	}
	n.messagePipe.Process(ctx, &bus.Message{Topic: bus.DataPlaneResponseTopic, Data: dpResponse})
}
