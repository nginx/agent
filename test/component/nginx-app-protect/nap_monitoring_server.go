package nginx_app_protect

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"

	pb "github.com/nginx/agent/sdk/v2/proto"
	events "github.com/nginx/agent/sdk/v2/proto/events"
)

type IngestionServerTest struct {
	channel        chan *events.EventReport
	receivedEvents map[string]*events.Event
	grpcServer     *grpc.Server
}

func NewIngestionServerTest(serverAddr string) (*IngestionServerTest, error) {
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	ingestionServer := &IngestionServerTest{
		channel:        make(chan *events.EventReport),
		receivedEvents: make(map[string]*events.Event),
		grpcServer:     grpcServer,
	}

	pb.RegisterIngesterServer(grpcServer, ingestionServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	return ingestionServer, nil
}

func (s *IngestionServerTest) StreamMetricsReport(pb.Ingester_StreamMetricsReportServer) error {
	return fmt.Errorf("not implemented")
}

func (s *IngestionServerTest) StreamEventReport(stream pb.Ingester_StreamEventReportServer) error {
	for {
		eventReport, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&types.Empty{})
		}
		if err != nil {
			return err
		}
		s.channel <- eventReport
	}
}

func (s *IngestionServerTest) Run(ctx context.Context) {
	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
		return
	case eventsReport := <-s.channel:
		for _, event := range eventsReport.Events {
			s.receivedEvents[event.GetSecurityViolationEvent().SupportID] = event
		}
	}
}

func (s *IngestionServerTest) ReceivedEvent(supportID string) (event *events.Event, found bool) {
	event, found = s.receivedEvents[supportID]
	return
}
