package services

import (
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	log "github.com/sirupsen/logrus"
)

type IngesterGRPCService struct {
	sync.RWMutex
	fromClient chan *proto.MetricsReport
	reports    []*proto.MetricsReport
}

func NewIngesterSvc() *IngesterGRPCService {
	return &IngesterGRPCService{
		fromClient: make(chan *proto.MetricsReport, 100),
	}
}

func (grpcService *IngesterGRPCService) StreamMetricsReport(stream proto.Ingester_StreamMetricsReportServer) error {
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

func (grpcService *IngesterGRPCService) GetMetrics() []*proto.MetricsReport {
	return grpcService.reports
}
