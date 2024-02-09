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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/dataplane"
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

func (*DataplaneServer) Close() {}

func (*DataplaneServer) Info() *bus.Info {
	return &bus.Info{
		Name: "dataplane-server",
	}
}

func (dps *DataplaneServer) Process(msg *bus.Message) {
	switch {
	case msg.Topic == bus.InstancesTopic:
		if newInstances, ok := msg.Data.([]*instances.Instance); ok {
			dps.instances = newInstances
		}
	case msg.Topic == bus.InstanceConfigUpdateCompleteTopic:
		if configStatus, ok := msg.Data.(*instances.ConfigurationStatus); ok {
			dps.configurationStatues[configStatus.GetInstanceId()] = configStatus
		}
	}
}

func (*DataplaneServer) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
		bus.InstanceConfigUpdateCompleteTopic,
	}
}

func (dps *DataplaneServer) run(_ context.Context) {
	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.Use(sloggin.NewWithConfig(dps.logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	dataplane.RegisterHandlersWithOptions(server, dps, dataplane.GinServerOptions{BaseURL: "/api/v1"})

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
// nolint: revive // Get func not returning value
func (dps *DataplaneServer) GetInstances(ctx *gin.Context) {
	slog.Debug("Get instances request")

	response := []dataplane.Instance{}

	for _, instance := range dps.instances {
		response = append(response, dataplane.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.GetType().String())),
			Version:    &instance.Version,
		})
	}

	slog.Debug("Got instances", "instances", response)

	ctx.JSON(http.StatusOK, response)
}

// PUT /instances/{instanceID}/configurations
func (dps *DataplaneServer) UpdateInstanceConfiguration(ctx *gin.Context, instanceID string) {
	correlationID := uuid.New().String()
	slog.Debug("Update instance configuration request", "correlationID", correlationID, "instanceID", instanceID)

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

			dps.configurationStatues[instanceID] = &instances.ConfigurationStatus{
				InstanceId:    instanceID,
				CorrelationId: correlationID,
				Status:        instances.Status_IN_PROGRESS,
				LastUpdated:   timestamppb.Now(),
				Message:       "Instance configuration update in progress",
			}

			ctx.JSON(http.StatusOK, dataplane.CorrelationId{CorrelationId: &correlationID})
		} else {
			slog.Debug(
				"Unable to update instance configuration",
				"instanceID", instanceID,
				"correlationID", correlationID,
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
func (dps *DataplaneServer) GetInstanceConfigurationStatus(ctx *gin.Context, instanceID string) {
	status := dps.getConfigurationStatus(instanceID)

	if status != nil {
		responseBody := &dataplane.ConfigurationStatus{
			CorrelationId: &status.CorrelationId,
			LastUpdated:   toPtr(status.GetLastUpdated().AsTime()),
			Message:       &status.Message,
			Status:        mapStatusEnums(status.GetStatus().String()),
		}

		ctx.JSON(http.StatusOK, responseBody)
	} else {
		slog.Debug("Unable to get instance configuration status", "instanceID", instanceID)
		ctx.JSON(http.StatusNotFound, dataplane.ErrorResponse{Message: "Unable to find configuration status"})
	}
}

func (dps *DataplaneServer) getConfigurationStatus(instanceID string) *instances.ConfigurationStatus {
	return dps.configurationStatues[instanceID]
}

func (dps *DataplaneServer) getInstance(instanceID string) *instances.Instance {
	for _, instance := range dps.instances {
		if instance.GetInstanceId() == instanceID {
			return instance
		}
	}

	return nil
}

func mapTypeEnums(typeString string) dataplane.InstanceType {
	if typeString == instances.Type_NGINX.String() {
		return dataplane.NGINX
	}

	return dataplane.CUSTOM
}

func mapStatusEnums(typeString string) *dataplane.ConfigurationStatusType {
	if typeString == instances.Status_SUCCESS.String() {
		return toPtr(dataplane.SUCCESS)
	} else if typeString == instances.Status_FAILED.String() {
		return toPtr(dataplane.FAILED)
	} else if typeString == instances.Status_ROLLBACK_FAILED.String() {
		return toPtr(dataplane.ROLLBACKFAILED)
	}

	return toPtr(dataplane.INPROGESS)
}

func toPtr[T any](value T) *T {
	return &value
}
