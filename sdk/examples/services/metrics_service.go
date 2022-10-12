package services

import (
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	log "github.com/sirupsen/logrus"
)

type MetricsGrpcService struct {
	sync.RWMutex
	fromClient chan *proto.MetricsReport
	reports    []*proto.MetricsReport
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

func (grpcService *MetricsGrpcService) GetMetrics() []*proto.MetricsReport {
	return grpcService.reports
}
