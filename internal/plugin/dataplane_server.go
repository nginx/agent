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
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/internal/service"

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
		address         string
		logger          *slog.Logger
		instanceService service.InstanceServiceInterface
		instances       []*instances.Instance
		messagePipe     bus.MessagePipeInterface
		server          net.Listener
	}
)

func NewDataplaneServer(dataplaneServerParameters *DataplaneServerParameters) *DataplaneServer {
	if dataplaneServerParameters.instanceService == nil {
		dataplaneServerParameters.instanceService = service.NewInstanceService()
	}

	return &DataplaneServer{
		address:         fmt.Sprintf("%s:%d", dataplaneServerParameters.Host, dataplaneServerParameters.Port),
		logger:          dataplaneServerParameters.Logger,
		instanceService: dataplaneServerParameters.instanceService,
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
	if msg.Topic == bus.INSTANCES_TOPIC {
		var ok bool
		instancesResp, ok := msg.Data.([]*instances.Instance)
		if !ok {
			slog.Error("unable to cast message payload to instances.Instance", "payload", msg.Data)
			return
		}
		dps.instances = instancesResp
	}
}

func (dps *DataplaneServer) Subscriptions() []string {
	return []string{
		bus.INSTANCES_TOPIC,
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
func (dps *DataplaneServer) GetInstances(ctx *gin.Context) {
	slog.Debug("get instances request")

	response := []common.Instance{}

	for _, instance := range dps.instances {
		response = append(response, common.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.Type.String())),
			Version:    &instance.Version,
		})
	}

	slog.Debug("got instances", "instances", response)

	ctx.JSON(http.StatusOK, response)
}

// PUT /instances/{instanceId}/configurations
func (dps *DataplaneServer) UpdateInstanceConfiguration(ctx *gin.Context, instanceId string) {
	correlationId := uuid.New().String()
	slog.Debug("update instance configuration request", "correlationId", correlationId, "instanceId", instanceId)

	var request dataplane.UpdateInstanceConfigurationJSONRequestBody
	if err := ctx.Bind(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, "")
		return
	}

	if request.Location == nil {
		ctx.JSON(http.StatusBadRequest, common.ErrorResponse{Message: "missing location field in request body"})
	} else {
		instance := dps.getInstance(instanceId)
		if instance != nil {
			request := &model.InstanceConfigUpdateRequest{
				Instance:      dps.getInstance(instanceId),
				Location:      *request.Location,
				CorrelationId: correlationId,
			}
			dps.messagePipe.Process(&bus.Message{Topic: bus.INSTANCE_CONFIG_UPDATE_REQUEST_TOPIC, Data: request})
			ctx.JSON(http.StatusOK, dataplane.CorrelationId{CorrelationId: &correlationId})
		} else {
			slog.Debug("unable to update instance configuration", "instanceId", instanceId, "correlationId", correlationId)
			ctx.JSON(http.StatusNotFound, common.ErrorResponse{Message: fmt.Sprintf("Unable to find instance %s", instanceId)})
		}
	}
}

func (dps *DataplaneServer) getInstance(instanceId string) *instances.Instance {
	for _, instance := range dps.instances {
		if instance.GetInstanceId() == instanceId {
			return instance
		}
	}
	return nil
}

func mapTypeEnums(typeString string) common.InstanceType {
	if typeString == instances.Type_NGINX.String() {
		return common.NGINX
	}
	return common.CUSTOM
}

func toPtr[T any](value T) *T {
	return &value
}
