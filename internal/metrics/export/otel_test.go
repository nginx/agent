// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package export

import (
	"context"
	"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	sdk "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonV1 "go.opentelemetry.io/proto/otlp/common/v1"
	metricsV1 "go.opentelemetry.io/proto/otlp/metrics/v1"
	resV1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics/source/prometheus"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	grpcProtocol = "tcp"
	grpcEndpoint = "127.0.0.1:4317"
)

type OTelCollector struct {
	sdk.UnimplementedMetricsServiceServer
	results      []*metricsV1.ResourceMetrics
	resultsMutex *sync.RWMutex
}

func NewOTelCollector() *OTelCollector {
	return &OTelCollector{
		results:      []*metricsV1.ResourceMetrics{},
		resultsMutex: &sync.RWMutex{},
	}
}

func (oc *OTelCollector) Export(ctx context.Context, request *sdk.ExportMetricsServiceRequest) (
	*sdk.ExportMetricsServiceResponse, error,
) {
	oc.resultsMutex.Lock()
	defer oc.resultsMutex.Unlock()

	oc.results = append(oc.results, request.GetResourceMetrics()...)

	return &sdk.ExportMetricsServiceResponse{}, nil
}

func (oc *OTelCollector) GetResults() []*metricsV1.ResourceMetrics {
	oc.resultsMutex.RLock()
	defer oc.resultsMutex.RUnlock()

	return oc.results
}

func TestGRPCExporter_Constructor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exporter, err := NewGRPCExporter(ctx, types.GetAgentConfig().Metrics.OTelExporter.GRPC)
	require.NoError(t, err)
	require.NotNil(t, exporter)
}

// nolint: revive
func TestOTelExporter_Constructor(t *testing.T) {
	ctx := context.Background()

	id := "agent-unique-id"
	serviceName := "agent-test"
	converterFunc := prometheus.ConvertPrometheus

	expRes, err := resource.New(ctx,
		// Keep the default detectors
		resource.WithTelemetrySDK(),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNamespaceKey.String(serviceNamespace),
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceInstanceIDKey.String(id),
		),
	)
	require.NoError(t, err)

	exporter, err := NewOTelExporter(ctx, types.GetAgentConfig(), serviceName, id, converterFunc)
	require.NoError(t, err)
	assert.Equal(t, types.GetAgentConfig(), exporter.conf)
	assert.NotNil(t, exporter.intExp)
	assert.NotNil(t, exporter.convert)
	assert.NotNil(t, &exporter.bufferMutex)
	assert.NotNil(t, exporter.sink)
	assert.Empty(t, exporter.buffer)
	assert.NotNil(t, exporter.res)
	assert.Equal(t, expRes, exporter.res)
	assert.Equal(t, model.OTel, exporter.Type())

	t.Run("misconfiguration-errors", func(tt *testing.T) {
		testCases := []struct {
			name        string
			confModFunc func(*config.Config) *config.Config
			isErr       bool
			expErr      string
		}{
			{
				name: "Test 1: GRPC is nil",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics.OTelExporter.GRPC = nil

					return c
				},
				isErr:  true,
				expErr: "gRPC configuration missing",
			},
			{
				name: "Test 2: OTelExporter is nil",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics.OTelExporter = nil

					return c
				},
				isErr:  true,
				expErr: "OTel Exporter configuration missing",
			},
			{
				name: "Test 3: Metrics is nil",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics = nil

					return c
				},
				isErr:  true,
				expErr: "metrics configuration missing",
			},
			{
				name: "Test 4: Buffer length is a negative value",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics.OTelExporter.BufferLength = -1

					return c
				},
				isErr: false,
			},
			{
				name: "Test 5: Export retry count is a negative value",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics.OTelExporter.ExportRetryCount = -1

					return c
				},
				isErr: false,
			},
			{
				name: "Test 6: Export interval is a negative value",
				confModFunc: func(c *config.Config) *config.Config {
					c.Metrics.OTelExporter.ExportInterval = -1

					return c
				},
				isErr: false,
			},
		}

		for _, test := range testCases {
			tt.Run(test.name, func(ttt *testing.T) {
				c := test.confModFunc(types.GetAgentConfig())

				exporter, err = NewOTelExporter(ctx, c, serviceName, id, converterFunc)
				if test.isErr {
					require.Error(ttt, err)
					require.Nil(ttt, exporter)
					assert.Contains(ttt, err.Error(), test.expErr)
				} else {
					require.NoError(ttt, err)
					require.NotEmpty(t, exporter)
				}
			})
		}
	})
}

func TestOTelExporter_Sink(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	id := "agent-unique-id"
	serviceName := "agent-test"
	converterFunc := prometheus.ConvertPrometheus

	exporter, err := NewOTelExporter(ctx, types.GetAgentConfig(), serviceName, id, converterFunc)
	require.NoError(t, err)

	// Start listening for entries.
	go exporter.StartSink(ctx)

	inputEntry := model.DataEntry{
		Name:        "test-metric",
		Description: "this describes the metric",
		Type:        model.Counter,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "test-metric",
				Labels: map[string]string{
					"label-name": "label-value",
				},
				Value: int64(42),
			},
		},
	}

	expMetricData := metricdata.Metrics{
		Name:        inputEntry.Name,
		Description: inputEntry.Description,
		Unit:        "",
	}

	err = exporter.Export(inputEntry)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	assert.Len(t, exporter.getBuffer(), 1)

	bufferValue := exporter.getBuffer()[0]
	assert.Equal(t, expMetricData.Name, bufferValue.Name)
	assert.Equal(t, expMetricData.Description, bufferValue.Description)
	assert.Equal(t, expMetricData.Unit, bufferValue.Unit)
	assert.NotNil(t, bufferValue.Data)

	actualSum, ok := bufferValue.Data.(metricdata.Sum[int64])
	assert.True(t, ok)

	expSum := &metricdata.Sum[int64]{
		IsMonotonic: true,
		Temporality: metricdata.DeltaTemporality,
		DataPoints: []metricdata.DataPoint[int64]{
			{
				Attributes: attribute.NewSet(attribute.KeyValue{
					Key: "label-name", Value: attribute.StringValue("label-value"),
				}),
				Value: int64(42),
			},
		},
	}

	// The actual `Data` has a timestamp field, meaning we have to compare field-by-field instead of calling Equal().
	assert.Equal(t, expSum.IsMonotonic, actualSum.IsMonotonic)
	assert.Equal(t, expSum.Temporality, actualSum.Temporality)
	assert.Equal(t, expSum.DataPoints[0].Attributes, actualSum.DataPoints[0].Attributes)
	assert.Equal(t, expSum.DataPoints[0].Value, actualSum.DataPoints[0].Value)
}

// nolint: gocognit, revive, dupl, maintidx
func TestOTelMetrics(t *testing.T) {
	ctx := context.Background()

	otelSdkVersion, err := helpers.GetRequiredModuleVersion(t, "go.opentelemetry.io/otel/sdk", 3)
	require.NoError(t, err)

	expResource := &resV1.Resource{
		Attributes: []*commonV1.KeyValue{
			{
				Key: "service.instance.id",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: "agent-unique-id",
					},
				},
			},
			{
				Key: "service.name",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: "Prometheus",
					},
				},
			},
			{
				Key: "service.namespace",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: "nginx",
					},
				},
			},
			{
				Key: "telemetry.sdk.language",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: "go",
					},
				},
			},
			{
				Key: "telemetry.sdk.name",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: "opentelemetry",
					},
				},
			},
			{
				Key: "telemetry.sdk.version",
				Value: &commonV1.AnyValue{
					Value: &commonV1.AnyValue_StringValue{
						StringValue: otelSdkVersion,
					},
				},
			},
		},
		DroppedAttributesCount: 0,
	}

	inputs := []model.DataEntry{
		{
			Name:        "nginx_ingress_controller_nginx_last_reload_milliseconds",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Duration in milliseconds of the last NGINX reload",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_nginx_last_reload_milliseconds",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(161),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_ingress_resources_total",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Number of handled ingress resources",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_ingress_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "master",
					},
					Value: int64(1),
				},
				{
					Name: "nginx_ingress_controller_ingress_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "minion",
					},
					Value: int64(2),
				},
				{
					Name: "nginx_ingress_controller_ingress_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "regular",
					},
					Value: int64(3),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_workqueue_queue_duration_seconds",
			Type:        model.Histogram,
			SourceType:  model.Prometheus,
			Description: "How long in seconds an item stays in workqueue before being processed",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "0.1",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "0.5",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "1",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "5",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "10",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "50",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "+Inf",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_sum",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					Value: float64(3.600000000000002),
				},
				{
					Name: "nginx_ingress_controller_workqueue_queue_duration_seconds_count",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					Value: int64(36),
				},
			},
		},
	}

	expectations := []*metricsV1.Metric{
		{
			Name:        "nginx_ingress_controller_nginx_last_reload_milliseconds",
			Description: "Duration in milliseconds of the last NGINX reload",
			Unit:        "",
			Data: &metricsV1.Metric_Gauge{
				Gauge: &metricsV1.Gauge{
					DataPoints: []*metricsV1.NumberDataPoint{
						{
							Attributes: []*commonV1.KeyValue{
								{
									Key: "class",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "nginx",
										},
									},
								},
							},
							StartTimeUnixNano: 0, // Expectations ignore time fields.
							TimeUnixNano:      0,
							Value: &metricsV1.NumberDataPoint_AsInt{
								AsInt: 161,
							},
							Exemplars: nil,
							Flags:     0,
						},
					},
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_ingress_resources_total",
			Description: "Number of handled ingress resources",
			Unit:        "",
			Data: &metricsV1.Metric_Sum{
				Sum: &metricsV1.Sum{
					DataPoints: []*metricsV1.NumberDataPoint{
						{
							Attributes: []*commonV1.KeyValue{
								{
									Key: "class",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "nginx",
										},
									},
								},
								{
									Key: "type",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "master",
										},
									},
								},
							},
							Value: &metricsV1.NumberDataPoint_AsInt{
								AsInt: 1,
							},
						},
						{
							Attributes: []*commonV1.KeyValue{
								{
									Key: "class",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "nginx",
										},
									},
								},
								{
									Key: "type",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "minion",
										},
									},
								},
							},
							Value: &metricsV1.NumberDataPoint_AsInt{
								AsInt: 2,
							},
						},
						{
							Attributes: []*commonV1.KeyValue{
								{
									Key: "class",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "nginx",
										},
									},
								},
								{
									Key: "type",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "regular",
										},
									},
								},
							},
							Value: &metricsV1.NumberDataPoint_AsInt{
								AsInt: 3,
							},
						},
					},
					AggregationTemporality: metricsV1.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA,
					IsMonotonic:            true,
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_workqueue_queue_duration_seconds",
			Description: "How long in seconds an item stays in workqueue before being processed",
			Unit:        "",
			Data: &metricsV1.Metric_Histogram{
				Histogram: &metricsV1.Histogram{
					DataPoints: []*metricsV1.HistogramDataPoint{
						{
							Attributes: []*commonV1.KeyValue{
								{
									Key: "class",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "nginx",
										},
									},
								},
								{
									Key: "name",
									Value: &commonV1.AnyValue{
										Value: &commonV1.AnyValue_StringValue{
											StringValue: "taskQueue",
										},
									},
								},
							},
							Count: 36,
							Sum:   toPtr(3.600000000000002),
							BucketCounts: []uint64{
								36, 36, 36, 36, 36, 36, 36,
							},
							ExplicitBounds: []float64{
								0.1, 0.5, 1.0, 5.0, 10.0, 50.0, math.Inf(1),
							},
						},
					},
					AggregationTemporality: metricsV1.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
				},
			},
		},
	}

	otelCollector := NewOTelCollector()
	opts := make([]grpc.ServerOption, 0)
	go startGRPCServer(t, opts, otelCollector)

	// Populate only relevant configs.
	c := types.GetAgentConfig()
	c.Metrics.OTelExporter.GRPC.Target = grpcEndpoint

	id := "agent-unique-id"
	expScope := &commonV1.InstrumentationScope{
		Name:                   "github.com/agent/v3",
		Version:                c.Version,
		Attributes:             nil,
		DroppedAttributesCount: 0,
	}

	exporter, err := NewOTelExporter(ctx, c, model.Prometheus.String(), id, prometheus.ConvertPrometheus)
	require.NoError(t, err)
	go exporter.StartSink(ctx)
	time.Sleep(500 * time.Millisecond)

	for _, inp := range inputs {
		err = exporter.Export(inp)
		require.NoError(t, err)
	}

	assert.Eventually(
		t, func() bool { return len(otelCollector.GetResults()) == 1 },
		c.Metrics.OTelExporter.ExportInterval+1*time.Second, 1*time.Second,
	)

	for i, act := range otelCollector.GetResults() {
		assert.Equal(t, expResource, act.GetResource())
		assert.NotEmpty(t, act.GetScopeMetrics())
		assert.Equal(t, expScope, act.GetScopeMetrics()[0].GetScope())

		assertMetrics(t,
			expectations, otelCollector.GetResults()[i].GetScopeMetrics()[0].GetMetrics(),
		)
	}
}

func startGRPCServer(t *testing.T, opts []grpc.ServerOption, collector *OTelCollector) {
	t.Helper()
	lis, err := net.Listen(grpcProtocol, grpcEndpoint)
	if err != nil {
		t.Fail()
	}

	grpcServer := grpc.NewServer(opts...)

	sdk.RegisterMetricsServiceServer(grpcServer, collector)
	err = grpcServer.Serve(lis)
	if err != nil {
		t.Fail()
	}

	stopGrpcServer(grpcServer)
}

func stopGrpcServer(server *grpc.Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		signal.Stop(sigs)
		server.Stop()
		time.Sleep(200 * time.Millisecond)
		done <- struct{}{}
	}()

	<-done
	server.GracefulStop()
}

func assertMetrics(t *testing.T, expected, actuals []*metricsV1.Metric) {
	t.Helper()

	for i, act := range actuals {
		exp := expected[i]

		assert.Equal(t, exp.GetName(), act.GetName())
		assert.Equal(t, exp.GetDescription(), act.GetDescription())
		assert.Equal(t, exp.GetUnit(), act.GetUnit())

		switch actData := act.GetData().(type) {
		case *metricsV1.Metric_Gauge:
			expData := exp.GetData()
			casted, ok := expData.(*metricsV1.Metric_Gauge)
			assert.True(t, ok)
			assertNumberPoints(t, casted.Gauge.GetDataPoints(), actData.Gauge.GetDataPoints())
		case *metricsV1.Metric_Histogram:
			expData := exp.GetData()
			casted, ok := expData.(*metricsV1.Metric_Histogram)
			assert.True(t, ok)
			assert.Equal(t, casted.Histogram.GetAggregationTemporality(), actData.Histogram.GetAggregationTemporality())
			assertHistogramPoints(t, casted.Histogram.GetDataPoints(), actData.Histogram.GetDataPoints())
		case *metricsV1.Metric_Sum:
			expData := exp.GetData()
			casted, ok := expData.(*metricsV1.Metric_Sum)
			assert.True(t, ok)
			assert.Equal(t, casted.Sum.GetAggregationTemporality(), actData.Sum.GetAggregationTemporality())
			assert.Equal(t, casted.Sum.GetIsMonotonic(), actData.Sum.GetIsMonotonic())
			assertNumberPoints(t, casted.Sum.GetDataPoints(), actData.Sum.GetDataPoints())
		}
	}
}

func assertNumberPoints(t *testing.T, expected, actuals []*metricsV1.NumberDataPoint) {
	t.Helper()

	for i, act := range actuals {
		exp := expected[i]
		assert.Equal(t, exp.GetValue(), act.GetValue())
		assert.Equal(t, exp.GetAttributes(), act.GetAttributes())
		assert.Equal(t, exp.GetFlags(), act.GetFlags())
		assert.Equal(t, exp.GetExemplars(), act.GetExemplars())
	}
}

// nolint: testifylint
func assertHistogramPoints(t *testing.T, expected, actuals []*metricsV1.HistogramDataPoint) {
	t.Helper()

	for i, act := range actuals {
		exp := expected[i]
		assert.Equal(t, exp.GetAttributes(), act.GetAttributes())
		assert.Equal(t, exp.GetBucketCounts(), act.GetBucketCounts())
		assert.Equal(t, exp.GetCount(), act.GetCount())
		assert.Equal(t, exp.GetExplicitBounds(), act.GetExplicitBounds())
		assert.Equal(t, exp.GetMax(), act.GetMax())
		assert.Equal(t, exp.GetMin(), act.GetMin())
		assert.Equal(t, exp.GetSum(), act.GetSum())
		assert.Equal(t, exp.GetFlags(), act.GetFlags())
		assert.Equal(t, exp.GetExemplars(), act.GetExemplars())
	}
}

func toPtr[T any](input T) *T {
	return &input
}
