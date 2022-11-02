package services

import (
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	f5_nginx_agent_sdk_events "github.com/nginx/agent/sdk/v2/proto/events"
	log "github.com/sirupsen/logrus"
)

type MetricsGrpcService struct {
	sync.RWMutex
	fromClient   chan *proto.MetricsReport
	reports      []*proto.MetricsReport
	eventReports []*f5_nginx_agent_sdk_events.EventReport
}

func NewMetricsService() *MetricsGrpcService {
	return &MetricsGrpcService{
		fromClient: make(chan *proto.MetricsReport, 100),
	}
}

func (grpcService *MetricsGrpcService) Stream(stream proto.MetricsService_StreamServer) error {
	log.Trace("Metrics Channel")

	for {
		report, err := stream.Recv()
		if err != nil {
			// recommend handling error
			log.Debugf("Error in recvHandle %v", err)
			break
		}
		log.Info("Got metrics")
		grpcService.reports = append(grpcService.reports, report)
		grpcService.fromClient <- report
	}
	return nil
}

func (grpcService *MetricsGrpcService) StreamEvents(stream proto.MetricsService_StreamEventsServer) error {
	log.Trace("Event Report Channel")

	for {
		report, err := stream.Recv()
		if err != nil {
			// recommend handling error
			log.Debugf("Error in recvHandle %v", err)
			break
		}
		log.Info("Got metrics")
		grpcService.eventReports = append(grpcService.eventReports, report)
	}
	return nil
}

func (grpcService *MetricsGrpcService) GetMetrics() []*proto.MetricsReport {
	return grpcService.reports
}

func (grpcService *MetricsGrpcService) GetEventReports() []*f5_nginx_agent_sdk_events.EventReport {
	return grpcService.eventReports
}
