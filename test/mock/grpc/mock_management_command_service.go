// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/gin-gonic/gin"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	sloggin "github.com/samber/slog-gin"
)

type CommandService struct {
	v1.UnimplementedCommandServiceServer
	server                       *gin.Engine
	connectionRequest            *v1.CreateConnectionRequest
	requestChan                  chan *v1.ManagementPlaneRequest
	dataPlaneResponses           []*v1.DataPlaneResponse
	updateDataPlaneStatusRequest *v1.UpdateDataPlaneStatusRequest
	dataPlaneResponsesMutex      sync.Mutex
	connectionMutex              sync.Mutex
	updateDataPlaneStatusMutex   sync.Mutex
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewCommandService() *CommandService {
	mgs := &CommandService{
		requestChan:                make(chan *v1.ManagementPlaneRequest),
		connectionMutex:            sync.Mutex{},
		updateDataPlaneStatusMutex: sync.Mutex{},
		dataPlaneResponsesMutex:    sync.Mutex{},
	}

	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	logger := slog.New(handler)

	server := gin.New()
	server.UseRawPath = true
	server.Use(sloggin.NewWithConfig(logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))

	server.GET("/api/v1/connection", func(c *gin.Context) {
		mgs.connectionMutex.Lock()
		defer mgs.connectionMutex.Unlock()

		if mgs.connectionRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"connectionRequest": mgs.connectionRequest,
			})
		}
	})

	server.GET("/api/v1/status", func(c *gin.Context) {
		mgs.updateDataPlaneStatusMutex.Lock()
		defer mgs.updateDataPlaneStatusMutex.Unlock()

		if mgs.updateDataPlaneStatusRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"updateDataPlaneStatusRequest": mgs.updateDataPlaneStatusRequest,
			})
		}
	})

	server.GET("/api/v1/responses", func(c *gin.Context) {
		mgs.dataPlaneResponsesMutex.Lock()
		defer mgs.dataPlaneResponsesMutex.Unlock()

		if mgs.dataPlaneResponses == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"dataPlaneResponse": mgs.dataPlaneResponses,
			})
		}
	})

	server.POST("/api/v1/requests", func(c *gin.Context) {
		request := v1.ManagementPlaneRequest{}
		data, err := io.ReadAll(c.Request.Body)
		if err != nil {
			slog.Error("error reading request body", "err", err)
			c.JSON(http.StatusBadRequest, nil)
			return
		}

		// protojson is needed to deal with the one of protos
		pb := protojson.UnmarshalOptions{DiscardUnknown: true}
		err = pb.Unmarshal(data, &request)
		if err != nil {
			c.JSON(http.StatusBadRequest, nil)
			return
		}

		mgs.requestChan <- &request

		c.JSON(http.StatusOK, &request)
	})

	mgs.server = server

	return mgs
}

func (mgs *CommandService) StartServer(listener net.Listener) {
	slog.Info("Starting mock management plane http server", "address", listener.Addr().String())
	err := mgs.server.RunListener(listener)
	if err != nil {
		slog.Error("Failed to start mock management plane http server", "error", err)
	}
}

func (mgs *CommandService) CreateConnection(
	_ context.Context,
	request *v1.CreateConnectionRequest) (
	*v1.CreateConnectionResponse,
	error,
) {
	slog.Debug("Create connection request", "request", request)

	if request == nil {
		return nil, errors.New("empty connection request")
	}

	mgs.connectionMutex.Lock()
	mgs.connectionRequest = request
	mgs.connectionMutex.Unlock()

	return &v1.CreateConnectionResponse{
		Response: &v1.CommandResponse{
			Status:  v1.CommandResponse_COMMAND_STATUS_OK,
			Message: "Success",
		},
		AgentConfig: request.GetAgent().GetInstanceConfig().GetAgentConfig(),
	}, nil
}

func (mgs *CommandService) UpdateDataPlaneStatus(
	_ context.Context,
	request *v1.UpdateDataPlaneStatusRequest) (
	*v1.UpdateDataPlaneStatusResponse,
	error,
) {
	slog.Debug("Update data plane status request", "request", request)

	if request == nil {
		return nil, errors.New("empty update data plane status request")
	}

	mgs.updateDataPlaneStatusMutex.Lock()
	mgs.updateDataPlaneStatusRequest = request
	mgs.updateDataPlaneStatusMutex.Unlock()

	return &v1.UpdateDataPlaneStatusResponse{}, nil
}

func (mgs *CommandService) UpdateDataPlaneHealth(
	_ context.Context,
	_ *v1.UpdateDataPlaneHealthRequest) (
	*v1.UpdateDataPlaneHealthResponse,
	error,
) {
	return &v1.UpdateDataPlaneHealthResponse{}, nil
}

func (mgs *CommandService) Subscribe(in v1.CommandService_SubscribeServer) error {
	for {
		slog.Info("Starting Subscribe")
		request := <-mgs.requestChan

		slog.Debug("Subscribe", "request", request)

		err := in.Send(request)
		if err != nil {
			slog.Error("Failed to send management request", "error", err)
		}

		dataPlaneResponse, err := in.Recv()
		if err != nil {
			slog.Error("Failed to receive data plane response", "error", err)
		} else {
			mgs.dataPlaneResponsesMutex.Lock()
			mgs.dataPlaneResponses = append(mgs.dataPlaneResponses, dataPlaneResponse)
			mgs.dataPlaneResponsesMutex.Unlock()
		}
	}
}
