// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/datasource/host"
	agentGrpc "github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const retryInterval = 5 * time.Second

type (
	GrpcClient struct {
		messagePipe          bus.MessagePipeInterface
		config               *config.Config
		conn                 *grpc.ClientConn
		isConnected          *atomic.Bool
		commandServiceClient v1.CommandServiceClient
		cancel               context.CancelFunc
		subscribeCancel      context.CancelFunc
		connectionMutex      sync.Mutex
		subscribeMutex       sync.Mutex
		fileServiceClient    v1.FileServiceClient
	}
)

func NewGrpcClient(agentConfig *config.Config) *GrpcClient {
	if agentConfig != nil && agentConfig.Command.Server.Type == "grpc" {
		if agentConfig.Common == nil {
			slog.Error("Invalid common configuration settings")
			return nil
		}

		isConnected := &atomic.Bool{}
		isConnected.Store(false)

		return &GrpcClient{
			config:          agentConfig,
			isConnected:     isConnected,
			connectionMutex: sync.Mutex{},
			subscribeMutex:  sync.Mutex{},
		}
	}

	return nil
}

func (gc *GrpcClient) Init(ctx context.Context, messagePipe bus.MessagePipeInterface) (err error) {
	var (
		subscribeCtx  context.Context
		grpcClientCtx context.Context
	)

	slog.InfoContext(ctx, "Starting grpc client plugin")
	gc.messagePipe = messagePipe

	serverAddr := net.JoinHostPort(
		gc.config.Command.Server.Host,
		fmt.Sprint(gc.config.Command.Server.Port),
	)

	grpcClientCtx, gc.cancel = context.WithTimeout(ctx, gc.config.Client.Timeout)
	slog.InfoContext(ctx, "Dialing grpc server", "server_addr", serverAddr)

	info := host.NewInfo()
	resourceID := info.ResourceID(ctx)

	gc.connectionMutex.Lock()
	gc.conn, err = grpc.DialContext(grpcClientCtx, serverAddr, agentGrpc.GetDialOptions(gc.config, resourceID)...)
	gc.connectionMutex.Unlock()

	gc.fileServiceClient = v1.NewFileServiceClient(gc.conn)

	if err != nil {
		return err
	}

	if gc.conn == nil || gc.conn.GetState() == connectivity.Shutdown {
		return errors.New("can't connect to server")
	}

	gc.commandServiceClient = v1.NewCommandServiceClient(gc.conn)

	gc.subscribeMutex.Lock()
	subscribeCtx, gc.subscribeCancel = context.WithCancel(ctx)
	gc.subscribeMutex.Unlock()

	go gc.subscribe(subscribeCtx)

	return nil
}

// revive cognitive complexity 13 due to the nil checks
// nolint: revive
func (gc *GrpcClient) subscribe(ctx context.Context) {
	var subscribeClient v1.CommandService_SubscribeClient
	var err error

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if subscribeClient == nil {
				subscribeClient, err = gc.commandServiceClient.Subscribe(ctx)
				if err != nil {
					slog.ErrorContext(ctx, "Unable to subscribe", "error", err)
					// this is temporary and will be changed in a followup PR use back off and allow the time to be set
					// but needed to add retry to stop the logs being spammed with errors
					time.Sleep(retryInterval)

					continue
				}
			}

			request, recErr := subscribeClient.Recv()
			if recErr != nil {
				slog.ErrorContext(ctx, "Error receiving messages", "err", recErr)
				subscribeClient = nil
				time.Sleep(retryInterval)

				continue
			}
			slog.DebugContext(ctx, "Subscribe received", "request", request)

			gc.ProcessRequest(ctx, request)
		}
	}
}

func (gc *GrpcClient) ProcessRequest(ctx context.Context, request *v1.ManagementPlaneRequest) {
	switch request.GetRequest().(type) {
	case *v1.ManagementPlaneRequest_ConfigApplyRequest:
		subCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, logger.GenerateCorrelationID())
		gc.messagePipe.Process(subCtx, &bus.Message{
			Topic: bus.InstanceConfigUpdateRequestTopic,
			Data:  request.GetRequest(),
		})
	default:
		slog.InfoContext(ctx, "Management plane request not implemented yet", "request_type", request.GetRequest())
	}
}

func (gc *GrpcClient) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing grpc client plugin")

	gc.subscribeMutex.Lock()
	if gc.subscribeCancel != nil {
		gc.subscribeCancel()
	}
	gc.subscribeMutex.Unlock()

	gc.connectionMutex.Lock()
	defer gc.connectionMutex.Unlock()

	if gc.conn != nil {
		err := gc.conn.Close()
		if err != nil && gc.cancel != nil {
			slog.ErrorContext(ctx, "Failed to gracefully close gRPC connection", "error", err)
			gc.cancel()
		}
	}

	return nil
}

func (gc *GrpcClient) GetFileOverview(ctx context.Context, request *v1.GetOverviewRequest) (*v1.FileOverview, error) {
	resp, err := gc.fileServiceClient.GetOverview(ctx, request)
	return resp.GetOverview(), err
}

func (gc *GrpcClient) GetFileContents(ctx context.Context, request *v1.GetFileRequest) (*v1.FileContents, error) {
	resp, err := gc.fileServiceClient.GetFile(ctx, request)

	return resp.GetContents(), err
}

func (gc *GrpcClient) Info() *bus.Info {
	return &bus.Info{
		Name: "grpc-client",
	}
}

// cognitive complexity of 14 dont think there is a way around that
// nolint: revive
func (gc *GrpcClient) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.ResourceTopic:
		if resource, ok := msg.Data.(*v1.Resource); ok {
			err := gc.createConnection(ctx, resource)
			if err != nil {
				return
			}

			err = gc.sendDataPlaneStatusUpdate(ctx, resource)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to send data plane status update", "error", err)
			}
		}
	case bus.InstanceHealthTopic:
		if instances, ok := msg.Data.([]*v1.InstanceHealth); ok {
			err := gc.sendDataPlaneHealthUpdate(ctx, instances)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to send data plane health status", "error", err)
			}
		}
	default:
		slog.DebugContext(ctx, "Unknown topic", "topic", msg.Topic)
	}
}

func (gc *GrpcClient) createConnection(ctx context.Context, resource *v1.Resource) error {
	if !gc.isConnected.Load() {
		req, err := gc.createConnectionRequest(resource)
		if err != nil {
			slog.Error("Creating connection request", "error", err)
			return err
		}

		reqCtx, reqCancel := context.WithTimeout(ctx, gc.config.Common.MaxElapsedTime)
		defer reqCancel()

		connectFn := func() (*v1.CreateConnectionResponse, error) {
			response, connectErr := gc.commandServiceClient.CreateConnection(reqCtx, req)

			validatedError := validateGrpcError(connectErr)
			if validatedError != nil {
				slog.ErrorContext(reqCtx, "Failed to create connection", "error", validatedError)

				return nil, validatedError
			}

			return response, nil
		}

		response, err := backoff.RetryWithData(connectFn, backoffHelpers.Context(reqCtx, gc.config.Common))
		if err != nil {
			return err
		}

		slog.DebugContext(ctx, "Connection created", "response", response)
		slog.DebugContext(ctx, "Agent connected")

		gc.isConnected.Store(true)
	}

	gc.messagePipe.Process(ctx, &bus.Message{
		Topic: bus.ConfigClientTopic,
		Data: &GrpcConfigClient{
			grpcOverviewFn:    gc.GetFileOverview,
			grpFileContentsFn: gc.GetFileContents,
		},
	})

	return nil
}

func (gc *GrpcClient) createConnectionRequest(resource *v1.Resource) (*v1.CreateConnectionRequest, error) {
	id, err := uuid.NewV7()
	if err != nil {
		slog.Error("Generating message id", "error", err)
		return nil, err
	}

	correlationID, err := uuid.NewUUID()
	if err != nil {
		slog.Error("Generating correlation id", "error", err)
		return nil, err
	}

	return &v1.CreateConnectionRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     id.String(),
			CorrelationId: correlationID.String(),
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}, nil
}

func (gc *GrpcClient) Subscriptions() []string {
	return []string{
		bus.ResourceTopic,
		bus.InstanceHealthTopic,
	}
}

// says sendDataPlaneHealthUpdate & sendDataPlaneStatusUpdate are duplicate code
// nolint: dupl
func (gc *GrpcClient) sendDataPlaneHealthUpdate(ctx context.Context, instanceHealths []*v1.InstanceHealth) error {
	if !gc.isConnected.Load() {
		slog.InfoContext(ctx, "gRPC client not connected yet. Skipping sending data plane health update")
		return nil
	}
	correlationID := logger.GetCorrelationID(ctx)

	request := &v1.UpdateDataPlaneHealthRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		InstanceHealths: instanceHealths,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, gc.config.Client.Timeout)
	defer backoffCancel()

	sendDataPlaneHealth := func() (*v1.UpdateDataPlaneHealthResponse, error) {
		slog.DebugContext(ctx, "Sending data plane health update request", "request", request)
		if gc.commandServiceClient == nil {
			return nil, errors.New("command service client is not initialized")
		}

		response, updateError := gc.commandServiceClient.UpdateDataPlaneHealth(ctx, request)

		validatedError := validateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update data plane health", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(sendDataPlaneHealth, backoffHelpers.Context(backOffCtx, gc.config.Common))
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, " UpdateDataPlaneHealth response ", "response", response)

	return nil
}

// nolint: dupl
func (gc *GrpcClient) sendDataPlaneStatusUpdate(
	ctx context.Context,
	resource *v1.Resource,
) error {
	if !gc.isConnected.Load() {
		slog.DebugContext(ctx, "gRPC client not connected yet. Skipping sending data plane status update")
		return nil
	}

	correlationID := logger.GetCorrelationID(ctx)

	request := &v1.UpdateDataPlaneStatusRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Resource: resource,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, gc.config.Client.Timeout)
	defer backoffCancel()

	sendDataPlaneStatus := func() (*v1.UpdateDataPlaneStatusResponse, error) {
		slog.DebugContext(ctx, "Sending data plane status update request", "request", request)
		if gc.commandServiceClient == nil {
			return nil, errors.New("command service client is not initialized")
		}

		response, updateError := gc.commandServiceClient.UpdateDataPlaneStatus(ctx, request)

		validatedError := validateGrpcError(updateError)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update data plane status", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(sendDataPlaneStatus, backoffHelpers.Context(backOffCtx, gc.config.Common))
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, " UpdateDataPlaneStatus response ", "response", response)

	return nil
}

func validateGrpcError(err error) error {
	if err != nil {
		if statusError, ok := status.FromError(err); ok {
			if statusError.Code() == codes.InvalidArgument {
				return backoff.Permanent(err)
			}
		}

		return err
	}

	return nil
}

type GrpcConfigClient struct {
	grpcOverviewFn    func(ctx context.Context, request *v1.GetOverviewRequest) (*v1.FileOverview, error)
	grpFileContentsFn func(ctx context.Context, request *v1.GetFileRequest) (*v1.FileContents, error)
}

func (gcc *GrpcConfigClient) GetOverview(
	ctx context.Context,
	request *v1.GetOverviewRequest,
) (*v1.FileOverview, error) {
	return gcc.grpcOverviewFn(ctx, request)
}

func (gcc *GrpcConfigClient) GetFile(ctx context.Context, request *v1.GetFileRequest) (*v1.FileContents, error) {
	return gcc.grpFileContentsFn(ctx, request)
}
