package mock

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/nginx/agent/sdk/v2/proto"
	events "github.com/nginx/agent/sdk/v2/proto/events"
)

type IngestionServerMock struct {
	channel        chan *events.EventReport
	receivedEvents map[string]*events.Event
	grpcServer     *grpc.Server
	logger         *log.Entry
}

func NewIngestionServerMock(serverAddr string) (*IngestionServerMock, error) {
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	ingestionServer := &IngestionServerMock{
		channel:        make(chan *events.EventReport),
		receivedEvents: make(map[string]*events.Event),
		grpcServer:     grpcServer,
		logger: log.WithFields(log.Fields{
			"extension": "mock-ingestion-server",
		}),
	}

	pb.RegisterMetricsServiceServer(grpcServer, ingestionServer)

	var grpcErr error
	go func() {
		grpcErr = grpcServer.Serve(listener)
	}()

	// Letting error be affected if there is any error while doing grpcServer.Serve()
	time.Sleep(time.Second)

	return ingestionServer, grpcErr
}

func (s *IngestionServerMock) Stream(pb.MetricsService_StreamServer) error {
	return fmt.Errorf("not implemented")
}

func (s *IngestionServerMock) StreamEvents(stream pb.MetricsService_StreamEventsServer) error {
	for {
		eventReport, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&types.Empty{})
		}
		if err != nil {
			return err
		}
		s.logger.Debugf("Got an eventReport: %v", eventReport)
		s.channel <- eventReport
	}
}

func (s *IngestionServerMock) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.grpcServer.Stop()
			return
		case eventsReport := <-s.channel:
			for _, event := range eventsReport.Events {
				s.logger.Debugf("Got an event with Support ID %v, storing it to receivedEvents", event.GetSecurityViolationEvent().SupportID)
				s.receivedEvents[event.GetSecurityViolationEvent().SupportID] = event
			}
		}
	}
}

func (s *IngestionServerMock) ReceivedEvent(supportID string) (event *events.Event, found bool) {
	event, found = s.receivedEvents[supportID]
	if !found {
		s.logger.Debug(
			fmt.Sprintf(
				"Could not find an event with Support ID %v, have receivedEvents: %v",
				supportID,
				s.receivedEvents,
			),
		)
	}
	return
}
