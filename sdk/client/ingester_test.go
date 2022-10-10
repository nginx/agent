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

func TestIngesterReporter_Server(t *testing.T) {
	ingesterReporterClient := NewIngesterClient()
	ingesterReporterClient.WithServer("test")

	assert.Equal(t, "test", ingesterReporterClient.Server())
}

func TestIngesterReporter_SendMetricsReport(t *testing.T) {
	grpcServer, ingesterReporterService, dialer := startIngesterReporterMockServer()

	ctx := context.TODO()

	ingesterReporterClient := createTestIngesterReporterClient(dialer)
	err := ingesterReporterClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		ingesterReporterClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = ingesterReporterClient.SendMetricsReport(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	select {
	case actual := <-ingesterReporterService.ingesterReporterHandler.metricReportStream:
		assert.Equal(t, "1234", actual.GetMeta().MessageId)
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from stream")
	}
}

func TestIngesterReporter_Connect_NoServer(t *testing.T) {
	ctx := context.TODO()

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	ingesterReporterClient := NewIngesterClient()
	ingesterReporterClient.WithServer("unknown")
	ingesterReporterClient.WithDialOptions(grpcDialOptions...)
	ingesterReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	err := ingesterReporterClient.Connect(ctx)
	assert.NotNil(t, err)
}

func TestIngesterReporter_Send_ServerDies(t *testing.T) {
	grpcServer, _, dialer := startIngesterReporterMockServer()

	ctx := context.TODO()

	ingesterReporterClient := createTestIngesterReporterClient(dialer)
	err := ingesterReporterClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		ingesterReporterClient.Close()
	}()

	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	err = ingesterReporterClient.SendMetricsReport(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.NotNil(t, err)
}

func TestIngesterReporter_Send_Reconnect(t *testing.T) {
	grpcServer, _, dialer := startIngesterReporterMockServer()

	ctx := context.TODO()

	ingesterReporterClient := createTestIngesterReporterClient(dialer)
	ingesterReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := ingesterReporterClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	grpcServer, ingesterReporterService, dialer := startIngesterReporterMockServer()
	ingesterReporterClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		ingesterReporterClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = ingesterReporterClient.SendMetricsReport(ctx, MessageFromMetrics(&proto.MetricsReport{
		Meta: &proto.Metadata{
			MessageId: "1234",
		},
	}))
	assert.Nil(t, err)

	select {
	case actual := <-ingesterReporterService.ingesterReporterHandler.metricReportStream:
		assert.Equal(t, "1234", actual.GetMeta().MessageId)
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from stream")
	}
}

type ingesterReporterHandlerFunc func(proto.Ingester_StreamMetricsReportServer, *sync.WaitGroup)

type ingesterReporterHandler struct {
	streamHandleFunc   ingesterReporterHandlerFunc
	metricReportStream chan *proto.MetricsReport
}

type mockIngesterReporterService struct {
	sync.RWMutex
	ingesterReporterHandler *ingesterReporterHandler
}

func (c *mockIngesterReporterService) StreamMetricsReport(stream proto.Ingester_StreamMetricsReportServer) error {
	wg := &sync.WaitGroup{}
	h := c.ensureIngesterReporterHandler()
	wg.Add(1)

	streamHandleFunc := h.streamHandleFunc
	if streamHandleFunc == nil {
		streamHandleFunc = h.streamHandle
	}

	go streamHandleFunc(stream, wg)

	wg.Wait()

	return nil
}

func (c *mockIngesterReporterService) ensureIngesterReporterHandler() *ingesterReporterHandler {
	c.RLock()
	if c.ingesterReporterHandler == nil {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.ingesterReporterHandler = &ingesterReporterHandler{}
		c.ingesterReporterHandler.metricReportStream = make(chan *proto.MetricsReport)
		return c.ingesterReporterHandler
	}
	defer c.RUnlock()
	return c.ingesterReporterHandler
}

func (h *ingesterReporterHandler) streamHandle(server proto.Ingester_StreamMetricsReportServer, wg *sync.WaitGroup) {
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

func startIngesterReporterMockServer() (*grpc.Server, *mockIngesterReporterService, func(context.Context, string) (net.Conn, error)) {
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(sdkGRPC.DefaultServerDialOptions...)
	ingesterReporterService := &mockIngesterReporterService{}
	ingesterReporterService.ingesterReporterHandler = ingesterReporterService.ensureIngesterReporterHandler()
	ingesterReporterService.ingesterReporterHandler.metricReportStream = make(chan *proto.MetricsReport)
	proto.RegisterIngesterServer(grpcServer, ingesterReporterService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			fmt.Printf("Error starting mock GRPC server: %v\n", err)
		}
	}()

	return grpcServer, ingesterReporterService, func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func createTestIngesterReporterClient(dialer func(context.Context, string) (net.Conn, error)) Ingester {
	ingesterReporterClient := NewIngesterClient()
	ingesterReporterClient.WithServer("bufnet")
	ingesterReporterClient.WithDialOptions(getDialOptions(dialer)...)
	ingesterReporterClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	return ingesterReporterClient
}
