/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics"
	"github.com/nginx/agent/v3/internal/metrics/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	metricSdk "go.opentelemetry.io/otel/sdk/metric"
)

const otelServiceName = "nginx-agent"

type (
	shutdownFunc func()

	DataSource interface {
		// Starts the data source and begins data collection at a regular interval. Metrics should be pushed to the
		// MessagePipe using the metrics topic, as they will be handled in the metrics plugin.
		Start(ctx context.Context, updateInterval time.Duration) error
		// Stops the data source from collecting data.
		Stop()
		// Getter for the type of data source.
		Type() string
	}

	// Tuple for a data source and its associated OTel Meter.
	MetricsSource struct {
		dataSource DataSource
		meter      metric.Meter
	}

	// The Metrics plugin. Discovers and owns the data sources that produce metrics for the Agent.
	Metrics struct {
		Sources       map[string]*MetricsSource    // key = DataSource.Type
		counterCache  map[string]metrics.DataPoint // key = DataEntry.Name
		conf          config.Config
		pipe          bus.MessagePipeInterface
		prod          *metrics.MetricsProducer
		shutdownFuncs []shutdownFunc
	}

	// The Metrics plugin can be configured with functional options.
	MetricsOption func(*Metrics) error
)

// Constructor for the Metrics plugin.
func NewMetrics(c config.Config, options ...MetricsOption) (*Metrics, error) {
	ctx, cancel := context.WithCancel(context.Background())

	prod := metrics.NewMetricsProducer(c.Version)
	meterProvider, err := metrics.NewMeterProvider(ctx, otelServiceName, *c.Metrics, m.prod)
	if err != nil {
		log.Printf("failed to create a meterProvider: %v", err)
	}

	// Sets the global meter provider.
	otel.SetMeterProvider(meterProvider)

	m := Metrics{
		Sources:       make(map[string]*MetricsSource, 0), // Key = data source type
		counterCache:  make(map[string]metrics.DataPoint, 0),
		conf:          c,
		pipe:          nil,
		prod:          prod,
		shutdownFuncs: []shutdownFunc{shutdownFunc(cancel)},
	}

	for _, opt := range options {
		err := opt(&m)
		if err != nil {
			return nil, fmt.Errorf("failed to apply metrics plugin option: %w", err)
		}
	}

	return &m, nil
}

// Initializes and starts the plugin. Required for the `Plugin` interface.
func (m *Metrics) Init(mp bus.MessagePipeInterface) error {
	m.pipe = mp
	go m.prod.StartListen(mp.Context())

	err := m.discoverSources()
	if err != nil {
		return fmt.Errorf("data source discovery failed: %w", err)
	}

	return m.startDataSources()
}

// Stops the metrics plugin. Required for the `Plugin` interface.
func (m *Metrics) Close() error {
	slog.Info("stopping all metrics data sources")
	for _, ms := range m.Sources {
		ms.dataSource.Stop()
	}

	for _, shutdown := range m.shutdownFuncs {
		shutdown()
	}

	return nil
}

// Returns info about the plugin. Required for the `Plugin` interface.
func (m *Metrics) Info() *bus.Info {
	return &bus.Info{
		Name: "metrics",
	}
}

// Processes an incoming Message Bus message in the plugin. Required for the `Plugin` interface.
func (m *Metrics) Process(msg *bus.Message) {
	switch msg.Topic {
	case bus.METRICS_TOPIC:
		entry, ok := msg.Data.(metrics.DataEntry)
		if !ok {
			slog.Warn("failed to convert metrics event to a data entry")
		}

		metricsSource, ok := m.Sources[entry.SourceType]
		if !ok {
			slog.Error("received message from unknown data source", "source", entry.SourceType)
		}

		switch entry.Type {
		case "gauge":
			m.addGauge(entry, metricsSource.meter)
		case "counter":
			m.addCounter(entry, metricsSource.meter)
		case "histogram":
			m.addHistogram(entry, metricsSource.meter)
		case "summary":
			slog.Debug("unhandled metrics event of type 'summary'")
		}

	case bus.OS_PROCESSES_TOPIC:
		slog.Debug("OS Processes have been updated")
		// TODO: We would need to add rediscovery logic here where any new data sources are added to the plugin's
		// sources slice.
	}
}

// Returns the list of topics the plugin is subscribed to. Required for the `Plugin` interface.
func (m *Metrics) Subscriptions() []string {
	return []string{
		bus.OS_PROCESSES_TOPIC,
		bus.METRICS_TOPIC,
	}
}

// Dynamically populates the `[]*MetricsSource` of `*Metrics`. This is currently a one-to-one mapping of each data type
// to its data source, can later be changed to a one-to-many.
func (m *Metrics) discoverSources() error {
	// Always initialized currently, should dynamically determine if we have a Prometheus source or not.
	// Data sources can also be passed in as options on plugin init.
	if _, ok := m.Sources[prometheus.DataSourceType]; !ok {
		m.Sources[prometheus.DataSourceType] = &MetricsSource{
			dataSource: prometheus.NewScraper(m.pipe),
			meter:      otel.GetMeterProvider().Meter(prometheus.DataSourceType, metric.WithInstrumentationVersion(m.conf.Version)),
		}
	}

	return nil
}

// Calls Start() on each data source.
func (m *Metrics) startDataSources() error {
	for _, metricsSrc := range m.Sources {
		go func(ds DataSource) {
			err := ds.Start(m.pipe.Context(), m.conf.Metrics.ReportInterval)
			if err != nil {
				// Probably need to figure out how to handle these errors better (e.g. with a `chan error`).
				slog.Error("failed to start metrics source", "error", err)
			}
		}(metricsSrc.dataSource)
	}
	return nil
}

func (m *Metrics) addGauge(de metrics.DataEntry, meter metric.Meter) {
	gauge := metrics.NewFloat64Gauge()

	for _, point := range de.Values {
		_, err := meter.Float64ObservableGauge(
			de.Name,
			metric.WithDescription(de.Description),
			metric.WithFloat64Callback(gauge.Callback),
		)
		if err != nil {
			slog.Error("failed to initialize OTel gauge", "error", err)
		}

		metricAttributes := []attribute.KeyValue{}
		if len(point.Labels) > 0 {
			for labelKey, labelValue := range point.Labels {
				metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
			}
		}
		gauge.Set(point.Value, attribute.NewSet(metricAttributes...))
	}
}

func (m *Metrics) addCounter(de metrics.DataEntry, meter metric.Meter) {
	counter, _ := meter.Float64Counter(
		de.Name,
		metric.WithDescription(de.Description),
	)

	for _, point := range de.Values {
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.Labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		if previousMetricValue, ok := m.counterCache[point.Name]; ok && point.Value > 0 {
			counter.Add(context.TODO(), point.Value-previousMetricValue.Value, metric.WithAttributeSet(attribute.NewSet(metricAttributes...)))
		} else {
			counter.Add(context.TODO(), point.Value, metric.WithAttributeSet(attribute.NewSet(metricAttributes...)))
		}

		m.counterCache[point.Name] = point
	}
}

func (m *Metrics) addHistogram(de metrics.DataEntry, meter metric.Meter) {
	histogram := metricdata.HistogramDataPoint[float64]{
		Bounds:       []float64{},
		BucketCounts: []uint64{},
		StartTime:    time.Now(),
		Time:         time.Now(),
	}

	for _, point := range de.Values {
		var bound string
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.Labels {
			if labelKey == "le" {
				bound = labelValue
			}
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		if strings.HasSuffix(point.Name, "_bucket") && bound != "" {
			histogram.BucketCounts = append(histogram.BucketCounts, uint64(point.Value))
			boundValue, _ := strconv.ParseFloat(bound, 64)
			histogram.Bounds = append(histogram.Bounds, boundValue)
		} else if strings.HasSuffix(point.Name, "_sum") {
			histogram.Sum = point.Value
		} else if strings.HasSuffix(point.Name, "_count") {
			histogram.Count = uint64(point.Value)
			histogram.Attributes = attribute.NewSet(metricAttributes...)
		}
	}

	m.prod.RecordMetrics(metricdata.Metrics{
		Name:        de.Name,
		Description: de.Description,
		Data: metricdata.Histogram[float64]{
			DataPoints:  []metricdata.HistogramDataPoint[float64]{histogram},
			Temporality: metricSdk.DefaultTemporalitySelector(metricSdk.InstrumentKindHistogram),
		},
	})
}

//##########################
//## Begin MetricsOptions ##
//##########################

// Appends a Metrics data source that will automatically collect metrics.
func WithDataSource(ds DataSource) MetricsOption {
	return func(m *Metrics) error {
		m.Sources[ds.Type()] = &MetricsSource{
			dataSource: ds,
			meter:      otel.GetMeterProvider().Meter(prometheus.DataSourceType, metric.WithInstrumentationVersion(m.conf.Version)),
		}
		return nil
	}
}

//########################
//## End MetricsOptions ##
//########################
