package metrics

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sdk "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	v1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/grpc"
)

var resourceMetrics []*v1.ResourceMetrics

func TestOTelMetrics(t *testing.T) {
	var tests = []struct {
		name     string
		metrics  map[string]bool
		delay    time.Duration
		endpoint string
		protocol string
		options []grpc.ServerOption
	}{
		{
			"nginx metrics test",
			map[string]bool{
				"nginx_active": false,
			},
			time.Second * 30,
			"0.0.0.0:4317",
			"tcp",
			[]grpc.ServerOption{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			t.Log("Running server")
			go func() {
				lis, err := net.Listen(test.protocol, test.endpoint)
				if err != nil {
					t.Fail()
				}
				grpcServer := grpc.NewServer(test.options...)

				sdk.RegisterMetricsServiceServer(grpcServer, OTelServer{})
				
				if err := grpcServer.Serve(lis); err != nil {
					t.Fail()
				}

				err = stopGrpcServer(grpcServer)
				if err != nil {
					t.Fail()
				}
			}()
			t.Log("Waiting for metrics")
			time.Sleep(test.delay)

			for _, resourceMetric := range resourceMetrics {
				for _, scopeMetric := range resourceMetric.ScopeMetrics {
					for _, metric := range scopeMetric.Metrics {
						_, ok := test.metrics[metric.GetName()]
						if ok {
							test.metrics[metric.GetName()] = true
						}
						metric.GetUnit()
					}
				}
			}
		
			for key, value := range test.metrics {
				assert.Truef(t, value, "NGINX %s metric not found", key)
			}
		})
	}
}

func stopGrpcServer(server *grpc.Server) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		signal.Stop(sigs)
		server.Stop()
		time.Sleep(200 * time.Millisecond)
		done <- true
	}()

	<-done
	server.GracefulStop()
	return nil
}

type OTelServer struct {
	sdk.UnimplementedMetricsServiceServer
}

func (OTelServer) Export(ctx context.Context, request *sdk.ExportMetricsServiceRequest) (*sdk.ExportMetricsServiceResponse, error) {
	resourceMetrics = append(resourceMetrics, request.GetResourceMetrics()...)
	return &sdk.ExportMetricsServiceResponse{}, nil
}
