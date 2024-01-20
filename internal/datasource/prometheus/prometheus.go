package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	metrics "github.com/nginx/agent/v3/internal/datasource/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	metricSdk "go.opentelemetry.io/otel/sdk/metric"
)

const defaultTarget = "http://192.168.59.101:30882/metrics"

// Used for deserializion of parsed Prometheus data.
type (
	// A single data point for an entry. An entry can have multiple points.
	DataPoint struct {
		Labels map[string]string
		Value  float64
	}

	// Represents a single entry for a Prometheus meter, is one of [counter, gauge, histogram, summary].
	DataEntry struct {
		Name        string
		Type        string
		Description string
		Values      []DataPoint
	}

	Scraper struct {
		meter                       metric.Meter
		producer                    *metrics.MetricsProducer
		previousCounterMetricValues map[string]DataPoint
		cancelFunc                  context.CancelFunc
	}
)

func NewScraper(meterProvider metric.MeterProvider, producer *metrics.MetricsProducer) *Scraper {
	meter := meterProvider.Meter("nginx-agent-prometheus-scraper", metric.WithInstrumentationVersion("v0.1"))
	return &Scraper{
		meter:                       meter,
		producer:                    producer,
		previousCounterMetricValues: map[string]DataPoint{},
	}
}

func (s *Scraper) Start(ctx context.Context, updateInterval time.Duration) error {
	slog.Info("prometheus data source booting up")
	ticker := time.NewTicker(updateInterval)

	cancellableCtx, cancelFunc := context.WithCancel(ctx)
	s.cancelFunc = cancelFunc

	// This should be discovered dynamically.
	targets := []string{defaultTarget}
	if value, ok := os.LookupEnv("PROMETHEUS_TARGETS"); ok {
		targets = strings.Split(value, ",")
	}

	for _, target := range targets {
		resp, err := http.DefaultClient.Get(strings.TrimSpace(target))
		if err != nil {
			return fmt.Errorf("could not find prometheus endpoint %s: %w", target, err)
		}

		log.Printf("found prometheus endpoint %v", resp.Request.URL)
	}

	for {
		select {
		case <-cancellableCtx.Done():
			return nil
		case <-ticker.C:
			for _, target := range targets {
				entries := scrapeEndpoint(strings.TrimSpace(target))

				for _, entry := range entries {
					switch entry.Type {
					case "gauge":
						s.addGauge(entry)
					case "counter":
						s.addCounter(entry)
					case "histogram":
						s.addHistogram(entry)
					case "summary":
						// no-op currently
					}
				}
			}
		}
	}
}

func (s *Scraper) Stop() {
	slog.Info("prometheus data source shutting down")
	s.cancelFunc()
}

func (s *Scraper) addGauge(de DataEntry) {
	gauge := metrics.NewFloat64Gauge()

	for _, point := range de.Values {
		_, err := s.meter.Float64ObservableGauge(
			de.Name,
			metric.WithDescription(de.Description),
			metric.WithFloat64Callback(gauge.Callback),
		)
		if err != nil {
			slog.Error("failed to initialize OTel gauge from Prometheus data", "error", err)
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

func (s *Scraper) addCounter(de DataEntry) {
	counter, _ := s.meter.Float64Counter(
		de.Name,
		metric.WithDescription(de.Description),
	)

	for _, point := range de.Values {
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.Labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		if previousMetricValue, ok := s.previousCounterMetricValues[de.Name]; ok && point.Value > 0 {
			counter.Add(context.TODO(), point.Value-previousMetricValue.Value, metric.WithAttributeSet(attribute.NewSet(metricAttributes...)))
		} else {
			counter.Add(context.TODO(), point.Value, metric.WithAttributeSet(attribute.NewSet(metricAttributes...)))
		}

		s.previousCounterMetricValues[de.Name] = point
	}
}

func (s *Scraper) addHistogram(de DataEntry) {
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

		if strings.HasSuffix(de.Name, "_bucket") && bound != "" {
			histogram.BucketCounts = append(histogram.BucketCounts, uint64(point.Value))
			boundValue, _ := strconv.ParseFloat(bound, 64)
			histogram.Bounds = append(histogram.Bounds, boundValue)
		} else if strings.HasSuffix(de.Name, "_sum") {
			histogram.Sum = point.Value
		} else if strings.HasSuffix(de.Name, "_count") {
			histogram.Count = uint64(point.Value)
			histogram.Attributes = attribute.NewSet(metricAttributes...)
		}
	}

	s.producer.RecordMetrics(metricdata.Metrics{
		Name:        de.Name,
		Description: de.Description,
		Data: metricdata.Histogram[float64]{
			DataPoints:  []metricdata.HistogramDataPoint[float64]{histogram},
			Temporality: metricSdk.DefaultTemporalitySelector(metricSdk.InstrumentKindHistogram),
		},
	})
}

func scrapeEndpoint(target string) []DataEntry {
	resp, err := http.Get(target)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	responseString := string(body)
	splitResponse := strings.Split(responseString, "\n")

	entries := make([]DataEntry, 0)

	currEntry := new(DataEntry)
	for _, line := range splitResponse {
		if strings.HasPrefix(line, "# HELP") {
			if currEntry.Name != "" {
				entries = append(entries, *currEntry)
				currEntry = new(DataEntry)
			}
			splitHelp := strings.SplitN(line, " ", 4)

			currEntry.Name = splitHelp[2]
			currEntry.Description = splitHelp[3]
		} else if strings.HasPrefix(line, "# TYPE") {
			splitType := strings.SplitN(line, " ", 4)
			currEntry.Type = splitType[3]
		} else if line != "" {
			labels := map[string]string{}
			splitMetric := strings.SplitN(line, " ", 3)
			nameAndLabels := splitMetric[0]
			splitNameAndLabels := strings.Split(nameAndLabels, "{")

			if len(splitNameAndLabels) > 1 {
				labelsJson := strings.ReplaceAll("{\""+splitNameAndLabels[1], "=", "\":")
				labelsJson = strings.ReplaceAll(labelsJson, ",", ",\"")
				err := json.Unmarshal([]byte(labelsJson), &labels)
				if err != nil {
					// Malformed labels will not be saved.
					slog.Warn("failed to parse Prometheus metric labels", "labels", splitNameAndLabels, "error", err)
				}
			}

			value, _ := strconv.ParseFloat(splitMetric[1], 64)
			currEntry.Values = append(currEntry.Values, DataPoint{
				Labels: labels,
				Value:  value,
			})

		}
	}

	log.Printf("%d metrics scraped from %s successfully", len(entries), target)
	return entries
}
