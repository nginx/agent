/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	http_api "github.com/nginx/agent/v3/api/http"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"

	sloggin "github.com/samber/slog-gin"
)

type (
	ErrorResponse struct {
		Message string `json:"message,omitempty"`
	}

	DataplaneServerParameters struct {
		Host            string
		Port            int
		Logger          *slog.Logger
		instanceService service.InstanceServiceInterface
	}

	DataplaneServer struct {
		address              string
		logger               *slog.Logger
		instanceService      service.InstanceServiceInterface
		instances            []*instances.Instance
		configurationStatues map[string]*instances.ConfigurationStatus
		messagePipe          bus.MessagePipeInterface
		server               net.Listener
	}
)

func NewDataplaneServer(dataplaneServerParameters *DataplaneServerParameters) *DataplaneServer {
	if dataplaneServerParameters.instanceService == nil {
		dataplaneServerParameters.instanceService = service.NewInstanceService()
	}

	return &DataplaneServer{
		address:              fmt.Sprintf("%s:%d", dataplaneServerParameters.Host, dataplaneServerParameters.Port),
		logger:               dataplaneServerParameters.Logger,
		instanceService:      dataplaneServerParameters.instanceService,
		configurationStatues: make(map[string]*instances.ConfigurationStatus),
	}
}

func (dps *DataplaneServer) Init(messagePipe bus.MessagePipeInterface) {
	dps.messagePipe = messagePipe
	go dps.run(messagePipe.Context())
}

func (dps *DataplaneServer) Close() {}

func (dps *DataplaneServer) Info() *bus.Info {
	return &bus.Info{
		Name: "dataplane-server",
	}
}

func (dps *DataplaneServer) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.INSTANCES_TOPIC:
		dps.instances = msg.Data.([]*instances.Instance)
	case msg.Topic == bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC:
		msgData := msg.Data.(*instances.ConfigurationStatus)
		dps.configurationStatues[msgData.GetInstanceId()] = msgData
	}
}

func (dps *DataplaneServer) Subscriptions() []string {
	return []string{
		bus.INSTANCES_TOPIC,
		bus.INSTANCE_CONFIG_UPDATE_COMPLETE_TOPIC,
	}
}

func (dps *DataplaneServer) run(ctx context.Context) {
	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.Use(sloggin.NewWithConfig(dps.logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	http_api.RegisterHandlersWithOptions(server, dps, http_api.GinServerOptions{BaseURL: "/api/v1"})

	slog.Info("Starting dataplane server", "address", dps.address)
	listener, err := net.Listen("tcp", dps.address)
	if err != nil {
		slog.Error("Startup of dataplane server failed", "error", err)
		return
	}

	dps.server = listener

	err = server.RunListener(listener)
	if err != nil {
		slog.Error("Startup of dataplane server failed", "error", err)
	}
}

// GET /instances
func (dps *DataplaneServer) GetInstances(ctx *gin.Context) {
	slog.Debug("Get instances request")

	response := []http_api.Instance{}

	for _, instance := range dps.instances {
		response = append(response, http_api.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.Type.String())),
			Version:    &instance.Version,
		})
	}

	slog.Debug("Got instances", "instances", response)

	ctx.JSON(http.StatusOK, response)
}

// PUT /instances/{instanceId}/configurations
func (dps *DataplaneServer) UpdateInstanceConfiguration(ctx *gin.Context, instanceId string) {
	correlationId := uuid.New().String()
	slog.Debug("Update instance configuration request", "correlationId", correlationId, "instanceId", instanceId)

	var request http_api.UpdateInstanceConfigurationJSONRequestBody
	if err := ctx.Bind(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, "")
		return
	}

	if request.Location == nil {
		ctx.JSON(http.StatusBadRequest, http_api.ErrorResponse{Message: "Missing location field in request body"})
	} else {
		instance := dps.getInstance(instanceId)
		if instance != nil {
			request := &model.InstanceConfigUpdateRequest{
				Instance:      dps.getInstance(instanceId),
				Location:      *request.Location,
				CorrelationId: correlationId,
			}

			dps.messagePipe.Process(&bus.Message{Topic: bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC, Data: request})

			dps.configurationStatues[instanceId] = &instances.ConfigurationStatus{
				InstanceId:    instanceId,
				CorrelationId: correlationId,
				Status:        instances.Status_IN_PROGRESS,
				LateUpdated:   timestamppb.Now(),
				Message:       "Instance configuration update in progress",
			}

			ctx.JSON(http.StatusOK, http_api.CorrelationId{CorrelationId: &correlationId})
		} else {
			slog.Debug("Unable to update instance configuration", "instanceId", instanceId, "correlationId", correlationId)
			ctx.JSON(http.StatusNotFound, http_api.ErrorResponse{Message: fmt.Sprintf("Unable to find instance %s", instanceId)})
		}
	}
}

// (GET /instances/{instanceId}/configurations/status)
func (dps *DataplaneServer) GetInstanceConfigurationStatus(ctx *gin.Context, instanceId string) {
	status := dps.getInstanceConfigurationStatus(instanceId)

	if status != nil {
		responseBody := &http_api.ConfigurationStatus{
			CorrelationId: &status.CorrelationId,
			LastUpdated:   toPtr(status.LateUpdated.AsTime()),
			Message:       &status.Message,
			Status:        mapStatusEnums(status.Status.String()),
		}

		ctx.JSON(http.StatusOK, responseBody)
	} else {
		slog.Debug("Unable to get instance configuration status", "instanceId", instanceId)
		ctx.JSON(http.StatusNotFound, http_api.ErrorResponse{Message: "Unable to find configuration status"})
	}
}

func (dps *DataplaneServer) getInstanceConfigurationStatus(instanceId string) *instances.ConfigurationStatus {
	return dps.configurationStatues[instanceId]
}

func (dps *DataplaneServer) getInstance(instanceId string) *instances.Instance {
	for _, instance := range dps.instances {
		if instance.GetInstanceId() == instanceId {
			return instance
		}
	}
	return nil
}

func mapTypeEnums(typeString string) http_api.InstanceType {
	if typeString == instances.Type_NGINX.String() {
		return http_api.NGINX
	}
	return http_api.CUSTOM
}

func mapStatusEnums(typeString string) *http_api.ConfigurationStatusType {
	if typeString == instances.Status_SUCCESS.String() {
		return toPtr(http_api.SUCCESS)
	} else if typeString == instances.Status_FAILED.String() {
		return toPtr(http_api.FAILED)
	} else if typeString == instances.Status_ROLLBACK_FAILED.String() {
		return toPtr(http_api.ROLLBACKFAILED)
	} else {
		return toPtr(http_api.INPROGESS)
	}
}

func toPtr[T any](value T) *T {
	return &value
}
