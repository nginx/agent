package client

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
)

func TestMetricReporter_Server(t *testing.T) {
	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer("test")

	assert.Equal(t, "test", metricReporterClient.Server())
}

func TestMetricReporter_Send(t *testing.T) {
	grpcServer, metricReporterService, dialer := startMetricReporterMockServer()

	ctx := context.TODO()

	metricReporterClient := createTestMetricReporterClient(dialer)
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		metricReporterClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = metricReporterClient.Send(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	select {
	case actual := <-metricReporterService.metricReporterHandler.metricReportStream:
		assert.Equal(t, "1234", actual.GetMeta().MessageId)
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from stream")
	}
}

func TestMetricReporter_Connect_NoServer(t *testing.T) {
	ctx := context.TODO()

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer("unknown")
	metricReporterClient.WithDialOptions(grpcDialOptions...)
	metricReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	err := metricReporterClient.Connect(ctx)
	assert.NotNil(t, err)
}

func TestMetricReporter_Send_ServerDies(t *testing.T) {
	grpcServer, _, dialer := startMetricReporterMockServer()

	ctx := context.TODO()

	metricReporterClient := createTestMetricReporterClient(dialer)
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		metricReporterClient.Close()
	}()

	if err := stopMockServer(grpcServer, dialer); err != nil {
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
	grpcServer, _, dialer := startMetricReporterMockServer()

	ctx := context.TODO()

	metricReporterClient := createTestMetricReporterClient(dialer)
	metricReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := metricReporterClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	grpcServer, metricReporterService, dialer := startMetricReporterMockServer()
	metricReporterClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		metricReporterClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = metricReporterClient.Send(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	select {
	case actual := <-metricReporterService.metricReporterHandler.metricReportStream:
		assert.Equal(t, "1234", actual.GetMeta().MessageId)
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from stream")
	}
}

type metricReporterHandlerFunc func(proto.MetricsService_StreamServer, *sync.WaitGroup)

type metricReporterHandler struct {
	streamHandleFunc   metricReporterHandlerFunc
	metricReportStream chan *proto.MetricsReport
}

type mockMetricReporterService struct {
	sync.RWMutex
	metricReporterHandler *metricReporterHandler
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

func (h *metricReporterHandler) streamHandle(server proto.MetricsService_StreamServer, wg *sync.WaitGroup) {
	for {
		cmd, err := server.Recv()
		fmt.Printf("Recv Metric Report: %v\n", cmd)
		if err != nil {
			fmt.Printf("Recv Metric Report: %v\n", err)
			return
		}
		h.metricReportStream <- cmd
	}
}

func startMetricReporterMockServer() (*grpc.Server, *mockMetricReporterService, func(context.Context, string) (net.Conn, error)) {
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(sdkGRPC.DefaultServerDialOptions...)
	metricReporterService := &mockMetricReporterService{}
	metricReporterService.metricReporterHandler = metricReporterService.ensureMetricReporterHandler()
	metricReporterService.metricReporterHandler.metricReportStream = make(chan *proto.MetricsReport)
	proto.RegisterMetricsServiceServer(grpcServer, metricReporterService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			fmt.Printf("Error starting mock GRPC server: %v\n", err)
		}
	}()

	return grpcServer, metricReporterService, func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func createTestMetricReporterClient(dialer func(context.Context, string) (net.Conn, error)) MetricReporter {
	metricReporterClient := NewMetricReporterClient()
	metricReporterClient.WithServer("bufnet")
	metricReporterClient.WithDialOptions(getDialOptions(dialer)...)
	metricReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	return metricReporterClient
}
