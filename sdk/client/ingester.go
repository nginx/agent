package client

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/nginx/agent/sdk/v2"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/nginx/agent/sdk/v2/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func NewIngesterClient() Ingester {
	return &ingesterReporter{
		connector:       newConnector(),
		backoffSettings: DefaultBackoffSettings,
	}
}

type ingesterReporter struct {
	*connector
	client          proto.IngesterClient
	metricsChannel  proto.Ingester_StreamMetricsReportClient
	ctx             context.Context
	mu              sync.Mutex
	backoffSettings BackoffSettings
}

func (r *ingesterReporter) WithInterceptor(interceptor interceptors.Interceptor) Client {
	r.connector.interceptors = append(r.connector.interceptors, interceptor)

	return r
}

func (r *ingesterReporter) WithClientInterceptor(interceptor interceptors.ClientInterceptor) Client {
	r.clientInterceptors = append(r.clientInterceptors, interceptor)

	return r
}

func (r *ingesterReporter) Connect(ctx context.Context) error {
	log.Debugf("Ingester reporter connecting to %s", r.server)

	r.ctx = ctx
	err := sdk.WaitUntil(
		r.ctx,
		r.backoffSettings.initialInterval,
		r.backoffSettings.maxInterval,
		r.backoffSettings.maxTimeout,
		r.createClient,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *ingesterReporter) createClient() error {
	log.Debug("Creating metric reporter client")
	r.mu.Lock()
	defer r.mu.Unlock()

	// Making sure that the previous client connection is closed before creating a new one
	if r.grpc != nil {
		err := r.grpc.Close()
		if err != nil {
			log.Warnf("Error closing old grpc connection: %v", err)
		}
	}

	grpc, err := sdkGRPC.NewGrpcConnectionWithContext(r.ctx, r.server, r.DialOptions())
	if err != nil {
		log.Errorf("Unable to create client connection to %s: %s", r.server, err)
		log.Infof("Metric reporter retrying to connect to %s", r.grpc.Target())
		return err
	}
	r.grpc = grpc

	r.client = proto.NewIngesterClient(r.grpc)

	channel, err := r.client.StreamMetricsReport(r.ctx)
	if err != nil {
		log.Warnf("Unable to create metrics channel: %s", err)
		log.Infof("Metric reporter retrying to connect to %s", r.grpc.Target())
		return err
	}
	r.metricsChannel = channel

	return nil
}

func (r *ingesterReporter) Close() (err error) {
	return r.closeConnection()
}

func (r *ingesterReporter) Server() string {
	return r.server
}

func (r *ingesterReporter) WithServer(s string) Client {
	r.server = s

	return r
}

func (r *ingesterReporter) DialOptions() []grpc.DialOption {
	return r.dialOptions
}

func (r *ingesterReporter) WithDialOptions(options ...grpc.DialOption) Client {
	r.dialOptions = append(r.dialOptions, options...)

	return r
}

func (r *ingesterReporter) WithBackoffSettings(backoffSettings BackoffSettings) Client {
	r.backoffSettings = backoffSettings
	return r
}

func (r *ingesterReporter) SendMetricsReport(ctx context.Context, message Message) error {
	var (
		report *proto.MetricsReport
		ok     bool
	)

	switch message.Classification() {
	case MsgClassificationMetric:
		if report, ok = message.Raw().(*proto.MetricsReport); !ok {
			return fmt.Errorf("IngesterReporter expected a metrics report message, but received %T", message.Data())
		}
	default:
		return fmt.Errorf("IngesterReporter expected a metrics report message, but received %T", message.Data())
	}

	err := sdk.WaitUntil(r.ctx, r.backoffSettings.initialInterval, r.backoffSettings.maxInterval, r.backoffSettings.sendMaxTimeout, func() error {
		if err := r.metricsChannel.Send(report); err != nil {
			return r.handleGrpcError("Metric Reporter Channel Send", err)
		}

		log.Tracef("IngesterReporter sent report %v", report)

		return nil
	})

	return err
}

func (r *ingesterReporter) closeConnection() error {
	err := r.metricsChannel.CloseSend()
	if err != nil {
		return err
	}
	return r.grpc.Close()
}

func (r *ingesterReporter) handleGrpcError(messagePrefix string, err error) error {
	if st, ok := status.FromError(err); ok {
		log.Errorf("%s: error communicating with %s, code=%s, message=%v", messagePrefix, r.grpc.Target(), st.Code().String(), st.Message())
	} else if err == io.EOF {
		log.Errorf("%s: server %s is not processing requests, code=%s, message=%v", messagePrefix, r.grpc.Target(), st.Code().String(), st.Message())
	} else {
		log.Errorf("%s: unknown grpc error while communicating with %s, %v", messagePrefix, r.grpc.Target(), err)
	}

	log.Infof("%s: retrying to connect to %s", messagePrefix, r.grpc.Target())
	_ = r.createClient()

	return err
}
