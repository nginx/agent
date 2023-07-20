/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/proto"
	f5_nginx_agent_sdk_events "github.com/nginx/agent/sdk/v2/proto/events"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
)

var grpcServerMetricsMutex = &sync.Mutex{}

func TestMetricReporter_Server(t *testing.T) {
	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer("test")

	assert.Equal(t, "test", metricReporterClient.Server())
}

func TestMetricReporter_Send(t *testing.T) {
	serverName, grpcServer, metricReporterService, dialer := startMetricReporterMockServer()

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	metricReporterClient := createTestMetricReporterClient(serverName, dialer)
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		metricReporterClient.Close()
		if err := stopMockMetricsServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	err = metricReporterClient.Send(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	go func() {
		defer wg.Done()
		select {
		case actual := <-metricReporterService.metricReporterHandler.metricReportStream:
			assert.Equal(t, "1234", actual.GetMeta().MessageId)
		case <-time.After(1 * time.Second):
			assert.Fail(t, "No message received from stream")
		}
	}()
	wg.Wait()
}

func TestMetricReporter_Connect_NoServer(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 200*time.Millisecond)

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer("unknown")
	metricReporterClient.WithDialOptions(grpcDialOptions...)
	metricReporterClient.WithBackoffSettings(backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  300 * time.Millisecond,
	})

	err := metricReporterClient.Connect(ctx)
	assert.NotNil(t, err)

	t.Cleanup(func() {
		metricReporterClient.Close()
		cncl()
	})
}

func TestMetricReporter_Send_ServerDies(t *testing.T) {
	serverName, grpcServer, _, dialer := startMetricReporterMockServer()

	ctx := context.Background()

	metricReporterClient := createTestMetricReporterClient(serverName, dialer)
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		metricReporterClient.Close()
	})

	if err := stopMockMetricsServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	err = metricReporterClient.Send(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.NotNil(t, err)
}

func TestMetricReporter_Send_Reconnect(t *testing.T) {
	serverName, grpcServer, _, dialer := startMetricReporterMockServer()

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	metricReporterClient := createTestMetricReporterClient(serverName, dialer)
	metricReporterClient.WithBackoffSettings(backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  10 * time.Second,
	})
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockMetricsServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	serverName, grpcServer, metricReporterService, dialer := startMetricReporterMockServer()
	metricReporterClient.WithDialOptions(getDialOptions(dialer)...)
	metricReporterClient.WithServer(serverName)

	t.Cleanup(func() {
		metricReporterClient.Close()
		if err := stopMockMetricsServer(ctx, grpcServer, dialer); err != nil {
			assert.Fail(t, "Unable to stop grpc server")
		}
	})

	err = metricReporterClient.Send(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	go func() {
		defer wg.Done()
		select {
		case actual := <-metricReporterService.metricReporterHandler.metricReportStream:
			assert.Equal(t, "1234", actual.GetMeta().MessageId)
		case <-time.After(1 * time.Second):
			assert.Fail(t, "No message received from stream")
		}
	}()
	wg.Wait()
}

type metricReporterHandlerFunc func(proto.MetricsService_StreamServer, *sync.WaitGroup)

type metricReporterHandler struct {
	streamHandleFunc   metricReporterHandlerFunc
	metricReportStream chan *proto.MetricsReport
}

type (
	eventReporterHandlerFunc func(proto.MetricsService_StreamEventsServer, *sync.WaitGroup)
	eventReporterHandler     struct {
		streamEventsHandleFunc eventReporterHandlerFunc
		eventReportStream      chan *f5_nginx_agent_sdk_events.EventReport
	}
)

type mockMetricReporterService struct {
	sync.RWMutex
	metricReporterHandler *metricReporterHandler
	eventReporterHandler  *eventReporterHandler
}

func (c *mockMetricReporterService) Stream(stream proto.MetricsService_StreamServer) error {
	wg := &sync.WaitGroup{}
	h := c.ensureMetricReporterHandler()
	wg.Add(1)

	streamHandleFunc := h.streamHandleFunc
	if streamHandleFunc == nil {
		streamHandleFunc = h.streamHandle
	}

	go streamHandleFunc(stream, wg)

	wg.Wait()

	return nil
}

func (c *mockMetricReporterService) StreamEvents(stream proto.MetricsService_StreamEventsServer) error {
	wg := &sync.WaitGroup{}
	h := c.ensureEventReporterHandler()
	wg.Add(1)

	streamEventsHandleFunc := h.streamEventsHandleFunc
	if streamEventsHandleFunc == nil {
		streamEventsHandleFunc = h.streamEventsHandle
	}

	go streamEventsHandleFunc(stream, wg)

	wg.Wait()

	return nil
}

func (c *mockMetricReporterService) ensureMetricReporterHandler() *metricReporterHandler {
	c.RLock()
	if c.metricReporterHandler == nil {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.metricReporterHandler = &metricReporterHandler{}
		c.metricReporterHandler.metricReportStream = make(chan *proto.MetricsReport)
		return c.metricReporterHandler
	}
	defer c.RUnlock()
	return c.metricReporterHandler
}

func (c *mockMetricReporterService) ensureEventReporterHandler() *eventReporterHandler {
	c.RLock()
	if c.eventReporterHandler == nil {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.eventReporterHandler = &eventReporterHandler{}
		c.eventReporterHandler.eventReportStream = make(chan *f5_nginx_agent_sdk_events.EventReport)
		return c.eventReporterHandler
	}
	defer c.RUnlock()
	return c.eventReporterHandler
}

func (h *metricReporterHandler) streamHandle(server proto.MetricsService_StreamServer, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		cmd, err := server.Recv()
		log.Debugf("Recv Metric Report: %v\n", cmd)
		if err != nil {
			log.Debugf("Recv Metric Report: %v\n", err)
			return
		}
		h.metricReportStream <- cmd
	}
}

func (h *eventReporterHandler) streamEventsHandle(server proto.MetricsService_StreamEventsServer, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		cmd, err := server.Recv()
		log.Debugf("Recv Event Report: %v\n", cmd)
		if err != nil {
			log.Debugf("Recv Event Report: %v\n", err)
			return
		}
		h.eventReportStream <- cmd
	}
}

func startMetricReporterMockServer() (string, *grpc.Server, *mockMetricReporterService, func(context.Context, string) (net.Conn, error)) {
	grpcServerMetricsMutex.Lock()
	defer grpcServerMetricsMutex.Unlock()
	serverName := fmt.Sprintf("%s_%s", uuid.New().String(), "bufnet")
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(sdkGRPC.DefaultServerDialOptions...)
	metricReporterService := &mockMetricReporterService{}
	metricReporterService.metricReporterHandler = metricReporterService.ensureMetricReporterHandler()
	metricReporterService.metricReporterHandler.metricReportStream = make(chan *proto.MetricsReport)
	proto.RegisterMetricsServiceServer(grpcServer, metricReporterService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Errorf("Error starting mock GRPC server: %v\n", err)
		}
	}()

	time.Sleep(200 * time.Millisecond)

	return serverName, grpcServer, metricReporterService, func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func createTestMetricReporterClient(serverName string, dialer func(context.Context, string) (net.Conn, error)) MetricReporter {
	grpcServerMetricsMutex.Lock()
	defer grpcServerMetricsMutex.Unlock()
	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer(serverName)
	metricReporterClient.WithDialOptions(getDialOptions(dialer)...)
	metricReporterClient.WithBackoffSettings(backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  300 * time.Millisecond,
	})

	return metricReporterClient
}

func stopMockMetricsServer(ctx context.Context, server *grpc.Server, dialer func(context.Context, string) (net.Conn, error)) error {
	grpcServerMetricsMutex.Lock()
	defer grpcServerMetricsMutex.Unlock()
	return stopMockServer(ctx, server, dialer)
}
