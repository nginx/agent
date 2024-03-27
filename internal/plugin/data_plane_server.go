// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	sloggin "github.com/samber/slog-gin"
)

type (
	ErrorResponse struct {
		Message string `json:"message,omitempty"`
	}

	DataPlaneServer struct {
		address           string
		logger            *slog.Logger
		instances         []*v1.Instance
		configEvents      map[string][]*instances.ConfigurationStatus
		messagePipe       bus.MessagePipeInterface
		server            *http.Server
		cancel            context.CancelFunc
		serverMutex       *sync.Mutex
		configEventsMutex *sync.Mutex
		instancesMutex    *sync.Mutex
	}
)

func NewDataPlaneServer(agentConfig *config.Config, logger *slog.Logger) *DataPlaneServer {
	address := net.JoinHostPort(agentConfig.DataPlaneAPI.Host, strconv.Itoa(agentConfig.DataPlaneAPI.Port))

	return &DataPlaneServer{
		address:           address,
		logger:            logger,
		configEvents:      make(map[string][]*instances.ConfigurationStatus),
		serverMutex:       &sync.Mutex{},
		configEventsMutex: &sync.Mutex{},
		instancesMutex:    &sync.Mutex{},
	}
}

func (dps *DataPlaneServer) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) error {
	slog.Info("Starting data plane server", "address", dps.address)

	dps.messagePipe = messagePipe

	var serverCtx context.Context
	serverCtx, dps.cancel = context.WithCancel(ctx)

	go dps.run(serverCtx)

	return nil
}

func (dps *DataPlaneServer) Close(ctx context.Context) error {
	slog.Debug("Closing data plane server plugin")

	// The context is used to inform the server it has, by default,
	// 5 seconds to finish the request it is currently handling
	serverShutdownCtx, cancel := context.WithTimeout(ctx, config.DefGracefulShutdownPeriod)
	defer cancel()

	dps.serverMutex.Lock()
	defer dps.serverMutex.Unlock()

	if err := dps.server.Shutdown(serverShutdownCtx); err != nil {
		slog.Error("Data plane server failed to shutdown", "error", err)
		dps.cancel()
	}

	slog.Info("Data plane server closed")

	return nil
}

func (*DataPlaneServer) Info() *bus.Info {
	return &bus.Info{
		Name: "dataplane-server",
	}
}

func (dps *DataPlaneServer) Process(_ context.Context, msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*v1.Instance); ok {
			dps.instancesMutex.Lock()
			dps.instances = newInstances
			dps.instancesMutex.Unlock()
		}
	case msg.Topic == bus.InstanceConfigUpdateTopic:
		if configStatus, ok := msg.Data.(*instances.ConfigurationStatus); ok {
			dps.updateEvents(configStatus)
		}
	}
}

func (dps *DataPlaneServer) updateEvents(configStatus *instances.ConfigurationStatus) {
	dps.configEventsMutex.Lock()
	defer dps.configEventsMutex.Unlock()

	instanceID := configStatus.GetInstanceId()
	if configStatus.GetStatus() == instances.Status_IN_PROGRESS {
		dps.configEvents[instanceID] = make([]*instances.ConfigurationStatus, 0)
		dps.configEvents[instanceID] = append(dps.configEvents[instanceID], configStatus)
	} else {
		dps.configEvents[instanceID] = append(dps.configEvents[instanceID], configStatus)
	}
}

func (*DataPlaneServer) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
		bus.InstanceConfigUpdateTopic,
	}
}

func (dps *DataPlaneServer) run(ctx context.Context) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(sloggin.NewWithConfig(dps.logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	dataplane.RegisterHandlersWithOptions(router, dps, dataplane.GinServerOptions{BaseURL: "/api/v1"})

	dps.serverMutex.Lock()
	dps.server = &http.Server{
		Addr:    dps.address,
		Handler: router,
	}
	dps.serverMutex.Unlock()

	go func() {
		if err := dps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to serve data plane server", "error", err)
			dps.cancel()
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()
}

// GET /instances
// nolint: revive // Get func not returning value
func (dps *DataPlaneServer) GetInstances(ctx *gin.Context) {
	slog.Debug("Get instances request")

	response := []dataplane.Instance{}

	dps.instancesMutex.Lock()
	defer dps.instancesMutex.Unlock()

	for _, instance := range dps.instances {
		response = append(response, dataplane.Instance{
			InstanceId: toPtr(instance.GetInstanceMeta().GetInstanceId()),
			Type:       toPtr(mapTypeEnums(instance.GetInstanceMeta().GetInstanceType().String())),
			Version:    toPtr(instance.GetInstanceMeta().GetVersion()),
			Meta:       toPtr(convertMeta(instance.GetInstanceConfig())),
		})
	}

	slog.Debug("Got instances", "instances", response)

	ctx.JSON(http.StatusOK, response)
}

// PUT /instances/{instanceID}/configurations
func (dps *DataPlaneServer) UpdateInstanceConfiguration(ctx *gin.Context, instanceID string) {
	correlationID := uuid.New().String()
	slog.Debug("Update instance configuration request", "correlation_id", correlationID, "instance_id", instanceID)

	var request dataplane.UpdateInstanceConfigurationJSONRequestBody
	if err := ctx.Bind(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, "")
		return
	}

	if request.Location == nil {
		ctx.JSON(http.StatusBadRequest, dataplane.ErrorResponse{Message: "Missing location field in request body"})
	} else {
		instance := dps.getInstance(instanceID)
		if instance != nil {
			request := &model.InstanceConfigUpdateRequest{
				Instance:      dps.getInstance(instanceID),
				Location:      *request.Location,
				CorrelationID: correlationID,
			}

			dps.messagePipe.Process(&bus.Message{Topic: bus.InstanceConfigUpdateRequestTopic, Data: request})

			ctx.JSON(http.StatusOK, dataplane.CorrelationId{CorrelationId: &correlationID})
		} else {
			slog.Debug(
				"Unable to update instance configuration",
				"instance_id", instanceID,
				"correlation_id", correlationID,
			)
			ctx.JSON(
				http.StatusNotFound,
				dataplane.ErrorResponse{Message: fmt.Sprintf("Unable to find instance %s", instanceID)},
			)
		}
	}
}

// (GET /instances/{instanceId}/configurations/status)
// nolint: revive // Get func not returning value
func (dps *DataPlaneServer) GetInstanceConfigurationStatus(ctx *gin.Context, instanceID string) {
	status := dps.getConfigurationStatus(instanceID)
	if status != nil {
		slog.Info("status", "", status)
		responseBody := &dataplane.ConfigurationStatus{
			InstanceId:    &instanceID,
			CorrelationId: &status[0].CorrelationId,
			Events:        convertConfigStatus(status),
		}

		ctx.JSON(http.StatusOK, responseBody)
	} else {
		slog.Debug("Unable to get instance configuration status", "instance_id", instanceID)
		ctx.JSON(http.StatusNotFound, dataplane.ErrorResponse{Message: "Unable to find configuration status"})
	}
}

func (dps *DataPlaneServer) getServerAddress() string {
	dps.serverMutex.Lock()
	defer dps.serverMutex.Unlock()

	return dps.server.Addr
}

func (dps *DataPlaneServer) getConfigurationStatus(instanceID string) []*instances.ConfigurationStatus {
	dps.configEventsMutex.Lock()
	defer dps.configEventsMutex.Unlock()

	return dps.configEvents[instanceID]
}

// nolint: revive
func (dps *DataPlaneServer) getInstances() []*v1.Instance {
	dps.instancesMutex.Lock()
	defer dps.instancesMutex.Unlock()

	return dps.instances
}

func (dps *DataPlaneServer) getInstance(instanceID string) *v1.Instance {
	dps.instancesMutex.Lock()
	defer dps.instancesMutex.Unlock()

	for _, instance := range dps.instances {
		if instance.GetInstanceMeta().GetInstanceId() == instanceID {
			return instance
		}
	}

	return nil
}

func mapTypeEnums(typeString string) dataplane.InstanceType {
	if typeString == v1.InstanceMeta_INSTANCE_TYPE_NGINX.String() {
		return dataplane.NGINX
	} else if typeString == v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS.String() {
		return dataplane.NGINXPLUS
	}

	return dataplane.CUSTOM
}

func mapStatusEnums(typeString string) *dataplane.StatusState {
	switch typeString {
	case instances.Status_IN_PROGRESS.String():
		return toPtr(dataplane.INPROGRESS)
	case instances.Status_SUCCESS.String():
		return toPtr(dataplane.SUCCESS)
	case instances.Status_FAILED.String():
		return toPtr(dataplane.FAILED)
	case instances.Status_ROLLBACK_IN_PROGRESS.String():
		return toPtr(dataplane.ROLLBACKINPROGRESS)
	case instances.Status_ROLLBACK_SUCCESS.String():
		return toPtr(dataplane.ROLLBACKSUCCESS)
	case instances.Status_ROLLBACK_FAILED.String():
		return toPtr(dataplane.ROLLBACKFAILED)
	}

	return toPtr(dataplane.INPROGRESS)
}

func convertConfigStatus(statuses []*instances.ConfigurationStatus) *[]dataplane.Event {
	dataplaneStatuses := []dataplane.Event{}
	for _, status := range statuses {
		dataplaneStatus := dataplane.Event{
			Timestamp: toPtr(status.GetTimestamp().AsTime()),
			Message:   &status.Message,
			Status:    mapStatusEnums(status.GetStatus().String()),
		}
		dataplaneStatuses = append(dataplaneStatuses, dataplaneStatus)
	}

	return &dataplaneStatuses
}

func convertMeta(instanceConfig *v1.InstanceConfig) dataplane.Instance_Meta {
	apiMeta := dataplane.Instance_Meta{}

	switch instanceConfig.GetConfig().(type) {
	case *v1.InstanceConfig_NginxConfig:
		err := apiMeta.MergeNginxMeta(
			dataplane.NginxMeta{
				Type:     dataplane.NGINXMETA,
				ConfPath: toPtr(instanceConfig.GetNginxConfig().GetConfigPath()),
				ExePath:  toPtr(instanceConfig.GetNginxConfig().GetBinaryPath()),
			},
		)
		if err != nil {
			slog.Warn("Unable to merge nginx meta", "error", err)
		}
	case *v1.InstanceConfig_NginxPlusConfig:
		err := apiMeta.MergeNginxMeta(
			dataplane.NginxMeta{
				Type:     dataplane.NGINXMETA,
				ConfPath: toPtr(instanceConfig.GetNginxPlusConfig().GetConfigPath()),
				ExePath:  toPtr(instanceConfig.GetNginxPlusConfig().GetBinaryPath()),
			},
		)
		if err != nil {
			slog.Warn("Unable to merge nginx meta", "error", err)
		}
	}

	return apiMeta
}

func toPtr[T any](value T) *T {
	return &value
}
