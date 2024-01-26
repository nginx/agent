/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	metricSdk "go.opentelemetry.io/otel/sdk/metric"
)

const (
	defaultTarget  = "http://127.0.0.1:30882/metrics"
	DataSourceType = "PROMETHEUS"
)

// Used for deserializion of parsed Prometheus data.
type Scraper struct {
	// Function called during Scraper shutdown.
	cancelFunc context.CancelFunc
	// Callback used to push metrics.
	reportCallback func(metricdata.Metrics)
}

func NewScraper() *Scraper {
	return &Scraper{
		reportCallback: nil,
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
				for _, metric := range entries {
					s.reportCallback(metric)
				}
			}
		}
	}
}

func (s *Scraper) Stop() {
	slog.Info("prometheus data source shutting down")
	s.cancelFunc()
}

func (s *Scraper) Type() string {
	return DataSourceType
}

func (s *Scraper) Register(callback func(metricdata.Metrics)) {
	s.reportCallback = callback
}

// Temporary tuple required for processing
type point struct {
	name   string
	labels map[string]string
	value  any
}

func scrapeEndpoint(target string) []metricdata.Metrics {
	resp, err := http.Get(target)
	if err != nil {
		slog.Error("GET to Prometheus HTTP scrape endpoint failed", "endpoint", target)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to parse Prometheus scrape endpoint response body")
	}

	responseString := string(body)
	splitResponse := strings.Split(responseString, "\n")

	entries := make([]metricdata.Metrics, 0)

	currEntry := metricdata.Metrics{}

	dataPoints := []point{}
	metricType := ""
	for _, line := range splitResponse {
		if strings.HasPrefix(line, "# HELP") {
			// If entry already has content, then we add it to the result slice and reset.
			if currEntry.Name != "" {
				err := addInstrument(&currEntry, metricType, dataPoints)
				if err != nil {
					slog.Warn("failed to add instrument for metric", "metric", currEntry.Name)
				}
				entries = append(entries, currEntry)
				currEntry, metricType, dataPoints = metricdata.Metrics{}, "", make([]point, 0)
			}
			splitHelp := strings.SplitN(line, " ", 4)
			currEntry.Name, currEntry.Description = splitHelp[2], splitHelp[3]
		} else if strings.HasPrefix(line, "# TYPE") {
			splitType := strings.SplitN(line, " ", 4)
			metricType = splitType[3]
		} else if line != "" {
			labels := map[string]string{}
			splitMetric := strings.SplitN(line, " ", 3)
			nameAndLabels := splitMetric[0]
			splitNameAndLabels := strings.Split(nameAndLabels, "{")

			pointName := splitNameAndLabels[0]
			var pointErr error
			// In Prometheus, the labels are wrapped in curly braces, so we transform their content into JSON format and
			// unmarshal them into a map.
			if len(splitNameAndLabels) > 1 {
				labelsJson := strings.ReplaceAll("{\""+splitNameAndLabels[1], "=", "\":")
				labelsJson = strings.ReplaceAll(labelsJson, ",", ",\"")
				err := json.Unmarshal([]byte(labelsJson), &labels)
				if err != nil {
					pointErr = errors.Join(pointErr, fmt.Errorf("failed to parse Prometheus data point labels [%s]: %w", splitNameAndLabels, err))
				}
			}

			// A value can either be a float or an int.
			var value any
			if strings.Contains(splitMetric[1], ".") {
				value, err = strconv.ParseFloat(splitMetric[1], 64)
			} else {
				value, err = strconv.ParseInt(splitMetric[1], 10, 64)
			}

			if err != nil {
				pointErr = errors.Join(pointErr, fmt.Errorf("failed to parse Prometheus data point value [%s]: %w", splitMetric[1], err))
			}

			if pointErr == nil {
				dataPoints = append(dataPoints, point{
					name:   pointName,
					labels: labels,
					value:  value,
				})
			} else {
				// Discard a data point that has malformed content, so we don't present dirty data to end users.
				slog.Warn("Prometheus data point parsing error, discarding point", "errors", pointErr)
			}

		}
	}

	err = addInstrument(&currEntry, metricType, dataPoints)
	if err != nil {
		slog.Warn("failed to add instrument for metric", "metric", currEntry.Name)
	}
	entries = append(entries, currEntry)

	slog.Debug("Prometheus endpoint scraped", "number of data entries", len(entries), "Prometheus URL", target)
	return entries
}

func addInstrument(metric *metricdata.Metrics, metricType string, points []point) error {
	if len(points) == 0 {
		return fmt.Errorf("metric [%s] had no data points", metric.Name)
	}

	var err error
	// Does Go have type inference where we could avoid some (or all) of these type switches?
	switch metricType {
	case "histogram":
		// Assume that all data points have same type.
		switch points[0].value.(type) {
		case int64:
			metric.Data, err = toHistogram[int64](points)
			if err != nil {
				return err
			}
		case float64:
			metric.Data, err = toHistogram[float64](points)
			if err != nil {
				return err
			}
		}
	case "counter":
		switch points[0].value.(type) {
		case int64:
			metric.Data, err = toCounter[int64](points)
			if err != nil {
				return err
			}
		case float64:
			metric.Data, err = toCounter[float64](points)
			if err != nil {
				return err
			}
		}
	case "gauge":
		switch points[0].value.(type) {
		case int64:
			metric.Data, err = toGauge[int64](points)
			if err != nil {
				return err
			}
		case float64:
			metric.Data, err = toGauge[float64](points)
			if err != nil {
				return err
			}
		}
	default:
		return nil
	}
	return nil
}

func toHistogram[N float64 | int64](points []point) (*metricdata.Histogram[N], error) {
	histogram := metricdata.HistogramDataPoint[N]{
		Bounds:       []float64{},
		BucketCounts: []uint64{},
		StartTime:    time.Now(),
		Time:         time.Now(),
	}

	for _, point := range points {
		var bound string
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.labels {
			if labelKey == "le" {
				bound = labelValue
			}
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		if strings.HasSuffix(point.name, "_bucket") && bound != "" {
			value, ok := point.value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not convert %v to uint64", point.value)
			}
			histogram.BucketCounts = append(histogram.BucketCounts, value)
			boundValue, _ := strconv.ParseFloat(bound, 64)
			histogram.Bounds = append(histogram.Bounds, boundValue)
		} else if strings.HasSuffix(point.name, "_sum") {
			value, ok := point.value.(N)
			if !ok {
				return nil, fmt.Errorf("unsupported histogram value type %T", point.value)
			}
			histogram.Sum = value
		} else if strings.HasSuffix(point.name, "_count") {
			value, ok := point.value.(uint64)
			if !ok {
				return nil, fmt.Errorf("could not convert %v to uint64", point.value)
			}
			histogram.Count = uint64(value)
			histogram.Attributes = attribute.NewSet(metricAttributes...)
		}
	}

	return &metricdata.Histogram[N]{
		DataPoints:  []metricdata.HistogramDataPoint[N]{histogram},
		Temporality: metricSdk.DefaultTemporalitySelector(metricSdk.InstrumentKindHistogram), // Not sure if Prometheus histograms are deltas or cumulatives.
	}, nil
}

func toCounter[N float64 | int64](points []point) (*metricdata.Sum[N], error) {
	datapoints := make([]metricdata.DataPoint[N], 0)

	for _, point := range points {
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		value, ok := point.value.(N)
		if !ok {
			// Do not return error for malformed data but instead ignore the data point.
			slog.Warn("data point has unexpected value [%T]", value)
			continue
		}

		datapoints = append(datapoints, metricdata.DataPoint[N]{
			Attributes: attribute.NewSet(metricAttributes...),
			Time:       time.Now(),
			Value:      value,
		})
	}

	return &metricdata.Sum[N]{
		DataPoints:  datapoints,
		Temporality: metricdata.DeltaTemporality,
		IsMonotonic: true,
	}, nil
}

func toGauge[N float64 | int64](points []point) (*metricdata.Gauge[N], error) {
	datapoints := make([]metricdata.DataPoint[N], 0)

	for _, point := range points {
		metricAttributes := []attribute.KeyValue{}
		for labelKey, labelValue := range point.labels {
			metricAttributes = append(metricAttributes, attribute.KeyValue{Key: attribute.Key(labelKey), Value: attribute.StringValue(labelValue)})
		}

		value, ok := point.value.(N)
		if !ok {
			// Do not return error for malformed data but instead ignore the data point.
			slog.Warn("data point has unexpected value [%T]", value)
			continue
		}

		datapoints = append(datapoints, metricdata.DataPoint[N]{
			Attributes: attribute.NewSet(metricAttributes...),
			Time:       time.Now(),
			Value:      value,
		})
	}

	return &metricdata.Gauge[N]{
		DataPoints: datapoints,
	}, nil
}
