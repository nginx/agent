// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/encoding/protojson"

	sloggin "github.com/samber/slog-gin"
)

type CommandService struct {
	v1.UnimplementedCommandServiceServer
	server                       *gin.Engine
	connectionRequest            *v1.CreateConnectionRequest
	requestChan                  chan *v1.ManagementPlaneRequest
	updateDataPlaneStatusRequest *v1.UpdateDataPlaneStatusRequest
	updateDataPlaneHealthRequest *v1.UpdateDataPlaneHealthRequest
	dataPlaneResponses           []*v1.DataPlaneResponse
	dataPlaneResponsesMutex      sync.Mutex
	updateDataPlaneStatusMutex   sync.Mutex
	updateDataPlaneHealthMutex   sync.Mutex
	connectionMutex              sync.Mutex
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewCommandService() *CommandService {
	cs := &CommandService{
		requestChan:                make(chan *v1.ManagementPlaneRequest),
		connectionMutex:            sync.Mutex{},
		updateDataPlaneStatusMutex: sync.Mutex{},
		updateDataPlaneHealthMutex: sync.Mutex{},
		dataPlaneResponsesMutex:    sync.Mutex{},
	}

	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	logger := slog.New(handler)

	cs.createServer(logger)

	return cs
}

func (cs *CommandService) StartServer(listener net.Listener) {
	slog.Info("Starting mock management plane http server", "address", listener.Addr().String())
	err := cs.server.RunListener(listener)
	if err != nil {
		slog.Error("Failed to start mock management plane http server", "error", err)
	}
}

func (cs *CommandService) CreateConnection(
	ctx context.Context,
	request *v1.CreateConnectionRequest) (
	*v1.CreateConnectionResponse,
	error,
) {
	slog.DebugContext(ctx, "Create connection request", "request", request)

	if request == nil {
		return nil, errors.New("empty connection request")
	}

	cs.connectionMutex.Lock()
	cs.connectionRequest = request
	cs.connectionMutex.Unlock()

	return &v1.CreateConnectionResponse{
		Response: &v1.CommandResponse{
			Status:  v1.CommandResponse_COMMAND_STATUS_OK,
			Message: "Success",
		},
		AgentConfig: request.GetResource().GetInstances()[0].GetInstanceConfig().GetAgentConfig(),
	}, nil
}

func (cs *CommandService) UpdateDataPlaneStatus(
	_ context.Context,
	request *v1.UpdateDataPlaneStatusRequest) (
	*v1.UpdateDataPlaneStatusResponse,
	error,
) {
	slog.Debug("Update data plane status request", "request", request)

	if request == nil {
		return nil, errors.New("empty update data plane status request")
	}

	cs.updateDataPlaneStatusMutex.Lock()
	cs.updateDataPlaneStatusRequest = request
	cs.updateDataPlaneStatusMutex.Unlock()

	return &v1.UpdateDataPlaneStatusResponse{}, nil
}

func (cs *CommandService) UpdateDataPlaneHealth(
	_ context.Context,
	request *v1.UpdateDataPlaneHealthRequest) (
	*v1.UpdateDataPlaneHealthResponse,
	error,
) {
	slog.Debug("Update data plane health request", "request", request)

	if request == nil {
		return nil, errors.New("empty update dataplane health request")
	}

	cs.updateDataPlaneHealthMutex.Lock()
	cs.updateDataPlaneHealthRequest = request
	cs.updateDataPlaneHealthMutex.Unlock()

	return &v1.UpdateDataPlaneHealthResponse{}, nil
}

func (cs *CommandService) Subscribe(in v1.CommandService_SubscribeServer) error {
	ctx := in.Context()

	go cs.listenForDataPlaneResponses(ctx, in)

	slog.InfoContext(ctx, "Starting Subscribe")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			request := <-cs.requestChan

			slog.DebugContext(ctx, "Subscribe", "request", request)

			err := in.Send(request)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to send management request", "error", err)
			}
		}
	}
}

func (cs *CommandService) listenForDataPlaneResponses(ctx context.Context, in v1.CommandService_SubscribeServer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			dataPlaneResponse, err := in.Recv()
			slog.DebugContext(ctx, "Received data plane response", "data_plane_response", dataPlaneResponse)
			if err != nil {
				slog.ErrorContext(ctx, "Failed to receive data plane response", "error", err)
				return
			}
			cs.dataPlaneResponsesMutex.Lock()
			cs.dataPlaneResponses = append(cs.dataPlaneResponses, dataPlaneResponse)
			cs.dataPlaneResponsesMutex.Unlock()
		}
	}
}

func (cs *CommandService) createServer(logger *slog.Logger) {
	cs.server = gin.New()
	cs.server.UseRawPath = true
	cs.server.Use(sloggin.NewWithConfig(logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))

	cs.addConnectionEndpoint()
	cs.addStatusEndpoint()
	cs.addHealthEndpoint()
	cs.addResponseAndRequestEndpoints()
}

func (cs *CommandService) addConnectionEndpoint() {
	cs.server.GET("/api/v1/connection", func(c *gin.Context) {
		cs.connectionMutex.Lock()
		defer cs.connectionMutex.Unlock()

		if cs.connectionRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(protojson.Format(cs.connectionRequest)), &data); err != nil {
				slog.Error("Failed to return connection", "error", err)
				c.JSON(http.StatusInternalServerError, nil)
			}
			c.JSON(http.StatusOK, data)
		}
	})
}

func (cs *CommandService) addStatusEndpoint() {
	cs.server.GET("/api/v1/status", func(c *gin.Context) {
		cs.updateDataPlaneStatusMutex.Lock()
		defer cs.updateDataPlaneStatusMutex.Unlock()

		if cs.updateDataPlaneStatusRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(protojson.Format(cs.updateDataPlaneStatusRequest)), &data); err != nil {
				slog.Error("Failed to return status", "error", err)
				c.JSON(http.StatusInternalServerError, nil)
			}
			c.JSON(http.StatusOK, data)
		}
	})
}

func (cs *CommandService) addHealthEndpoint() {
	cs.server.GET("/api/v1/health", func(c *gin.Context) {
		cs.updateDataPlaneHealthMutex.Lock()
		defer cs.updateDataPlaneHealthMutex.Unlock()

		if cs.updateDataPlaneHealthRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(protojson.Format(cs.updateDataPlaneHealthRequest)), &data); err != nil {
				slog.Error("Failed to return data plane health", "error", err)
				c.JSON(http.StatusInternalServerError, nil)
			}
			c.JSON(http.StatusOK, data)
		}
	})
}

func (cs *CommandService) addResponseAndRequestEndpoints() {
	cs.server.GET("/api/v1/responses", func(c *gin.Context) {
		cs.dataPlaneResponsesMutex.Lock()
		defer cs.dataPlaneResponsesMutex.Unlock()

		if cs.dataPlaneResponses == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			c.JSON(http.StatusOK, cs.dataPlaneResponses)
		}
	})

	cs.server.POST("/api/v1/requests", func(c *gin.Context) {
		request := v1.ManagementPlaneRequest{}
		body, err := io.ReadAll(c.Request.Body)
		slog.Debug("Received request", "body", body)
		if err != nil {
			slog.Error("Error reading request body", "err", err)
			c.JSON(http.StatusBadRequest, err)

			return
		}

		pb := protojson.UnmarshalOptions{DiscardUnknown: true}
		err = pb.Unmarshal(body, &request)
		if err != nil {
			slog.Error("Error unmarshalling request body", "err", err)
			c.JSON(http.StatusBadRequest, err)

			return
		}

		cs.requestChan <- &request

		c.JSON(http.StatusOK, &request)
	})
}
