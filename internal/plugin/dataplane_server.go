/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/http/common"
	"github.com/nginx/agent/v3/api/http/dataplane"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/nginx/agent/v3/internal/service"

	sloggin "github.com/samber/slog-gin"
)

const (
	internalServerError = "Internal Server Error"
)

type (
	ErrorResponse struct {
		Message string `json:"message,omitempty"`
	}

	DataplaneServerParameters struct {
		Address         string
		Logger          *slog.Logger
		instanceService service.InstanceServiceInterface
	}

	DataplaneServer struct {
		address         string
		logger          *slog.Logger
		instanceService service.InstanceServiceInterface
		messagePipe     *bus.MessagePipe
		server          net.Listener
		processes       []*os.Process
	}
)

func NewDataplaneServer(dataplaneServerParameters *DataplaneServerParameters) *DataplaneServer {
	if dataplaneServerParameters.instanceService == nil {
		dataplaneServerParameters.instanceService = service.NewInstanceService()
	}

	return &DataplaneServer{
		address:         dataplaneServerParameters.Address,
		logger:          dataplaneServerParameters.Logger,
		instanceService: dataplaneServerParameters.instanceService,
	}
}

func (dps *DataplaneServer) Init(messagePipe *bus.MessagePipe) {
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
	case msg.Topic == bus.OS_PROCESSES_TOPIC:
		newProcesses := msg.Data.([]*os.Process)
		dps.processes = newProcesses
		dps.instanceService.UpdateProcesses(newProcesses)
	}
}

func (dps *DataplaneServer) Subscriptions() []string {
	return []string{
		bus.OS_PROCESSES_TOPIC,
	}
}

func (dps *DataplaneServer) run(ctx context.Context) {
	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.Use(sloggin.NewWithConfig(dps.logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	dataplane.RegisterHandlersWithOptions(server, dps, dataplane.GinServerOptions{BaseURL: "/api/v1"})

	slog.Info("starting dataplane server", "address", dps.address)
	listener, err := net.Listen("tcp", dps.address)
	if err != nil {
		slog.Error("startup of dataplane server failed", "error", err)
		return
	}

	dps.server = listener

	err = server.RunListener(listener)
	if err != nil {
		slog.Error("startup of dataplane server failed", "error", err)
	}
}

// GET /instances
func (dps *DataplaneServer) GetInstances(ctx *gin.Context) {
	var statusCode int
	var responseBody any

	slog.Debug("get instances request")
	instances, err := dps.instanceService.GetInstances()
	slog.Debug("got instances", "instances", instances)

	response := []common.Instance{}

	for _, instance := range instances {
		response = append(response, common.Instance{
			InstanceId: &instance.InstanceId,
			Type:       toPtr(mapTypeEnums(instance.Type.String())),
			Version:    &instance.Version,
		})
	}

	if err != nil {
		statusCode = http.StatusInternalServerError
		responseBody = ErrorResponse{internalServerError}
	} else {
		statusCode = http.StatusOK
		responseBody = response

	}

	ctx.JSON(statusCode, responseBody)
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
