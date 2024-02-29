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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	sloggin "github.com/samber/slog-gin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	ErrorResponse struct {
		Message string `json:"message,omitempty"`
	}

	DataPlaneServer struct {
		address              string
		logger               *slog.Logger
		instances            []*instances.Instance
		configurationStatues map[string]*instances.ConfigurationStatus
		messagePipe          bus.MessagePipeInterface
		server               net.Listener
	}
)

func NewDataPlaneServer(agentConfig *config.Config, logger *slog.Logger) *DataPlaneServer {
	address := net.JoinHostPort(agentConfig.DataPlaneAPI.Host, strconv.Itoa(agentConfig.DataPlaneAPI.Port))

	return &DataPlaneServer{
		address:              address,
		logger:               logger,
		configurationStatues: make(map[string]*instances.ConfigurationStatus),
	}
}

func (dps *DataPlaneServer) Init(messagePipe bus.MessagePipeInterface) {
	dps.messagePipe = messagePipe
	go dps.run(messagePipe.Context())
}

func (*DataPlaneServer) Close() {}

func (*DataPlaneServer) Info() *bus.Info {
	return &bus.Info{
		Name: "dataplane-server",
	}
}

func (dps *DataPlaneServer) Process(msg *bus.Message) {
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

func (*DataPlaneServer) Subscriptions() []string {
	return []string{
		bus.InstancesTopic,
		bus.InstanceConfigUpdateCompleteTopic,
	}
}

func (dps *DataPlaneServer) run(_ context.Context) {
	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.Use(sloggin.NewWithConfig(dps.logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	dataplane.RegisterHandlersWithOptions(server, dps, dataplane.GinServerOptions{BaseURL: "/api/v1"})

	slog.Info("Starting data plane server", "address", dps.address)
	listener, err := net.Listen("tcp", dps.address)
	if err != nil {
		slog.Error("Startup of data plane server failed", "error", err)

		return
	}

	dps.server = listener

	err = server.RunListener(listener)
	if err != nil {
		slog.Error("Startup of data plane server failed", "error", err)
	}
}

// GET /instances
// nolint: revive // Get func not returning value
func (dps *DataPlaneServer) GetInstances(ctx *gin.Context) {
	slog.Debug("Get instances request")

	response := []dataplane.Instance{}

	for _, instance := range dps.instances {
		response = append(response, dataplane.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.GetType().String())),
			Version:    &instance.Version,
			Meta:       toPtr(convertMeta(instance.GetMeta())),
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
		responseBody := &dataplane.ConfigurationStatus{
			CorrelationId: &status.CorrelationId,
			LastUpdated:   toPtr(status.GetLastUpdated().AsTime()),
			Message:       &status.Message,
			Status:        mapStatusEnums(status.GetStatus().String()),
		}

		ctx.JSON(http.StatusOK, responseBody)
	} else {
		slog.Debug("Unable to get instance configuration status", "instance_id", instanceID)
		ctx.JSON(http.StatusNotFound, dataplane.ErrorResponse{Message: "Unable to find configuration status"})
	}
}

func (dps *DataPlaneServer) getConfigurationStatus(instanceID string) *instances.ConfigurationStatus {
	return dps.configurationStatues[instanceID]
}

func (dps *DataPlaneServer) getInstance(instanceID string) *instances.Instance {
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

func convertMeta(meta *instances.Meta) dataplane.Instance_Meta {
	apiMeta := dataplane.Instance_Meta{}

	if nginxMeta := meta.GetNginxMeta(); nginxMeta != nil {
		err := apiMeta.MergeNginxMeta(
			dataplane.NginxMeta{
				Type:     dataplane.NGINXMETA,
				ConfPath: &nginxMeta.ConfigPath,
				ExePath:  &nginxMeta.ExePath,
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
