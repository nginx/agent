// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	sloggin "github.com/samber/slog-gin"
)

type ManagementGrpcServer struct {
	v1.UnimplementedCommandServiceServer
	server             *gin.Engine
	connectionRequest  *v1.CreateConnectionRequest
	requestChan        chan *v1.ManagementPlaneRequest
	dataPlaneResponses []*v1.DataPlaneResponse
	mut                *sync.Mutex
}

func NewManagementGrpcServer() *ManagementGrpcServer {
	mgs := &ManagementGrpcServer{
		requestChan: make(chan *v1.ManagementPlaneRequest),
		mut:         &sync.Mutex{},
	}

	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	logger := slog.New(handler)

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.UseRawPath = true
	server.Use(sloggin.NewWithConfig(logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))

	server.GET("/api/v1/connection", func(c *gin.Context) {
		if mgs.connectionRequest == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"connectionRequest": mgs.connectionRequest,
			})
		}
	})

	server.GET("/api/v1/responses", func(c *gin.Context) {
		if mgs.dataPlaneResponses == nil {
			c.JSON(http.StatusNotFound, nil)
		} else {
			mgs.mut.Lock()

			c.JSON(http.StatusOK, gin.H{
				"dataPlaneResponse": mgs.dataPlaneResponses,
			})

			mgs.mut.Unlock()
		}
	})

	server.POST("/api/v1/requests", func(c *gin.Context) {
		var request *v1.ManagementPlaneRequest
		err := c.Bind(request)
		if err != nil {
			c.JSON(http.StatusBadRequest, nil)
			return
		}

		mgs.requestChan <- request

		c.JSON(http.StatusOK, request)
	})

	mgs.server = server

	return mgs
}

func (mgs *ManagementGrpcServer) StartServer(listener net.Listener) {
	slog.Info("Starting mock management plane gRPC server", "address", listener.Addr().String())
	err := mgs.server.RunListener(listener)
	if err != nil {
		slog.Error("Startup of mock management plane server failed", "error", err)
	}
}

func (mgs *ManagementGrpcServer) CreateConnection(
	_ context.Context,
	request *v1.CreateConnectionRequest) (
	*v1.CreateConnectionResponse,
	error,
) {
	slog.Debug("Create connection request", "request", request)

	if request == nil {
		return nil, errors.New("empty connection request")
	}

	mgs.connectionRequest = request

	return &v1.CreateConnectionResponse{
		Response: &v1.CommandResponse{
			Status:  v1.CommandResponse_COMMAND_STATUS_OK,
			Message: "Success",
		},
		AgentConfig: request.GetAgent().GetInstanceConfig().GetAgentConfig(),
	}, nil
}

func (mgs *ManagementGrpcServer) UpdateDataPlaneStatus(
	_ context.Context,
	_ *v1.UpdateDataPlaneStatusRequest) (
	*v1.UpdateDataPlaneStatusResponse,
	error,
) {
	return &v1.UpdateDataPlaneStatusResponse{}, nil
}

func (mgs *ManagementGrpcServer) UpdateDataPlaneHealth(
	ctx context.Context,
	in *v1.UpdateDataPlaneHealthRequest) (
	*v1.UpdateDataPlaneHealthResponse,
	error,
) {
	return &v1.UpdateDataPlaneHealthResponse{}, nil
}

func (mgs *ManagementGrpcServer) Subscribe(in v1.CommandService_SubscribeServer) error {
	for {
		request := <-mgs.requestChan

		slog.Debug("Subscribe", "request", request)

		err := in.Send(request)
		if err != nil {
			slog.Error("Failed to send management request", "err", err)
		}

		dataPlaneResponse, err := in.Recv()
		if err != nil {
			slog.Error("Failed to receive data plane response", "err", err)
		} else {
			mgs.mut.Lock()
			mgs.dataPlaneResponses = append(mgs.dataPlaneResponses, dataPlaneResponse)
			mgs.mut.Unlock()
		}
	}
}
