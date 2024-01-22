/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	met "github.com/nginx/agent/v3/internal/datasource/metric"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const prometheusTestDataFile = "prometheus_test_data.txt"

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate

//counterfeiter:generate -o ./mock_meter_provider.go ./ MeterProviderInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/prometheus mock_meter_provider.go | sed -e s\\/prometheus\\\\.\\/\\/g > mock_mock_meter_provider.go"
//go:generate mv mock_mock_meter_provider.go mock_meter_provider.go
type MeterProviderInterface interface {
	meterProvider()
	Meter(name string, opts ...metric.MeterOption) metric.Meter
}

func TestPrometheus_Scrape(t *testing.T) {
	testdata, err := os.ReadFile(prometheusTestDataFile)
	assert.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(testdata))
	})

	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	results := scrapeEndpoint(fakePrometheus.URL)
	assert.Len(t, results, 182)
}

func TestPrometheus_Constructor(t *testing.T) {
	mp := otel.GetMeterProvider()
	p := met.NewMetricsProducer()

	scraper := NewScraper(mp, p)
	assert.NotNil(t, scraper)

	assert.Equal(t, p, scraper.producer)
	assert.Equal(t, make(map[string]DataPoint), scraper.previousCounterMetricValues)
}

func TestPrometheus_Start(t *testing.T) {
	mp := otel.GetMeterProvider()
	p := met.NewMetricsProducer()

	scraper := NewScraper(mp, p)
	testdata, err := os.ReadFile(prometheusTestDataFile)
	assert.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(testdata))
	})

	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	err = os.Setenv("PROMETHEUS_TARGETS", fakePrometheus.URL)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var errChan chan error
	go func(ec chan error) {
		errChan <- scraper.Start(ctx, 1*time.Second)
	}(errChan)

	<-time.After(1500 * time.Millisecond)
	cancel()
	assert.Empty(t, errChan)
}

func TestPrometheus_Stop(t *testing.T) {
	mp := otel.GetMeterProvider()
	p := met.NewMetricsProducer()

	scraper := NewScraper(mp, p)
	testdata, err := os.ReadFile(prometheusTestDataFile)
	assert.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(testdata))
	})

	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	err = os.Setenv("PROMETHEUS_TARGETS", fakePrometheus.URL)
	assert.NoError(t, err)

	var errChan chan error
	go func(ec chan error) {
		errChan <- scraper.Start(context.Background(), 1*time.Second)
	}(errChan)

	<-time.After(1500 * time.Millisecond)
	scraper.Stop()
	assert.Empty(t, errChan)
}

func TestPrometheus_Type(t *testing.T) {
	mp := otel.GetMeterProvider()
	p := met.NewMetricsProducer()

	scraper := NewScraper(mp, p)
	assert.Equal(t, "PROMETHEUS", scraper.Type())
}
