// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package prometheus

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	prometheusTestDataFile = "prometheus_test_data.txt"
)

type (
	testPoint struct {
		labels map[string]string
		value  float64
	}

	testInput struct {
		name        string
		desc        string
		points      []testPoint
		histBuckets []float64
		histValues  map[float64]int
		histLabels  map[string]string
	}
)

func TestPrometheus_Constructor(t *testing.T) {
	targets := []string{"target_1", "target_2"}

	scraper := NewScraper(targets)
	assert.NotNil(t, scraper)

	require.Equal(t, targets, scraper.endpoints)
	assert.Nil(t, scraper.cancelFunc)
}

func TestPrometheus_Produce(t *testing.T) {
	testdata, err := os.ReadFile(prometheusTestDataFile)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, string(testdata))
	})

	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	targets := []string{fakePrometheus.URL}

	s := NewScraper(targets)

	results, err := s.Produce(context.Background())

	time.Sleep(2 * time.Second)
	require.NoError(t, err)
	assert.Len(t, results, 183)

	firstEntry := model.DataEntry{
		Name:        "go_gc_duration_seconds",
		Description: "A summary of the pause duration of garbage collection cycles.",
		Type:        model.Summary,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name: "go_gc_duration_seconds",
				Labels: map[string]string{
					"quantile": "0",
				},
				Value: 0.00003849,
			},
			{
				Name: "go_gc_duration_seconds",
				Labels: map[string]string{
					"quantile": "0.25",
				},
				Value: 0.000171217,
			},
			{
				Name: "go_gc_duration_seconds",
				Labels: map[string]string{
					"quantile": "0.5",
				},
				Value: 0.000252162,
			},
			{
				Name: "go_gc_duration_seconds",
				Labels: map[string]string{
					"quantile": "0.75",
				},
				Value: 0.0008172,
			},
			{
				Name: "go_gc_duration_seconds",
				Labels: map[string]string{
					"quantile": "1",
				},
				Value: 0.003638701,
			},
			{
				Name:   "go_gc_duration_seconds_sum",
				Labels: nil,
				Value:  0.012623016,
			},
			{
				Name:   "go_gc_duration_seconds_count",
				Labels: nil,
				Value:  int64(20),
			},
		},
	}

	assert.Equal(t, firstEntry, results[0])

	secondEntry := model.DataEntry{
		Name:        "go_goroutines",
		Description: "Number of goroutines that currently exist.",
		Type:        model.Gauge,
		SourceType:  model.Prometheus,
		Values: []model.DataPoint{
			{
				Name:   "go_goroutines",
				Labels: nil,
				Value:  int64(30),
			},
		},
	}
	assert.Equal(t, secondEntry, results[1])
}

// nolint: maintidx
func TestPrometheusIntegration_Gauges(t *testing.T) {
	input := []testInput{
		{
			name: "nginx_ingress_controller_nginx_last_reload_milliseconds",
			desc: "Duration in milliseconds of the last NGINX reload",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 161,
				},
			},
		},
		{
			name: "nginx_ingress_controller_nginx_last_reload_status",
			desc: "Status of the last NGINX reload",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 1,
				},
			},
		},
		{
			name: "nginx_ingress_controller_transportserver_resources_total",
			desc: "Number of handled TransportServer resources",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "passthrough",
					},
					value: 0,
				},
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "tcp",
					},
					value: 0,
				},
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "udp",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_virtualserver_resources_total",
			desc: "Number of handled VirtualServer resources",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_virtualserverroute_resources_total",
			desc: "Number of handled VirtualServerRoute resources",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_nginx_worker_processes_total",
			desc: "Number of NGINX worker processes",
			points: []testPoint{
				{
					labels: map[string]string{
						"class":      "nginx",
						"generation": "current",
					},
					value: 12,
				},
				{
					labels: map[string]string{
						"class":      "nginx",
						"generation": "old",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_workqueue_depth",
			desc: "Current depth of workqueue",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_active",
			desc: "Active client connections",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 1,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_reading",
			desc: "Connections where NGINX is reading the request header",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_waiting",
			desc: "Idle client connections",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_writing",
			desc: "Connections where NGINX is writing the response back to the client",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 1,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_up",
			desc: "Status of the last metric scrape",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 1,
				},
			},
		},
	}

	expectations := []model.DataEntry{
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
			Name:        "nginx_ingress_controller_nginx_last_reload_status",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Status of the last NGINX reload",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_nginx_last_reload_status",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(1),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_transportserver_resources_total",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Number of handled TransportServer resources",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_transportserver_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "passthrough",
					},
					Value: int64(0),
				},
				{
					Name: "nginx_ingress_controller_transportserver_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "tcp",
					},
					Value: int64(0),
				},
				{
					Name: "nginx_ingress_controller_transportserver_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "udp",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_virtualserver_resources_total",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Number of handled VirtualServer resources",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_virtualserver_resources_total",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_virtualserverroute_resources_total",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Number of handled VirtualServerRoute resources",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_virtualserverroute_resources_total",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_nginx_worker_processes_total",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Number of NGINX worker processes",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_nginx_worker_processes_total",
					Labels: map[string]string{
						"class":      "nginx",
						"generation": "current",
					},
					Value: int64(12),
				},
				{
					Name: "nginx_ingress_controller_nginx_worker_processes_total",
					Labels: map[string]string{
						"class":      "nginx",
						"generation": "old",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_workqueue_depth",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Current depth of workqueue",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_workqueue_depth",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_active",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Active client connections",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_active",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(1),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_reading",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Connections where NGINX is reading the request header",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_reading",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_waiting",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Idle client connections",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_waiting",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_writing",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Connections where NGINX is writing the response back to the client",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_writing",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(1),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_up",
			Type:        model.Gauge,
			SourceType:  model.Prometheus,
			Description: "Status of the last metric scrape",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_up",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(1),
				},
			},
		},
	}

	for _, inp := range input {
		gauge := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: inp.name,
			Help: inp.desc,
		}, getKeys(inp.points[0].labels))
		for _, point := range inp.points {
			gauge.With(point.labels).Add(point.value)
		}
		// It's important to reset the Prometheus instruments so they don't interfere with other tests.
		//nolint: revive
		defer gauge.Reset()
	}

	ctx := context.Background()
	s := httptest.NewServer(promhttp.Handler())
	defer s.Close()

	scraper := NewScraper([]string{s.URL})

	res, err := scraper.Produce(ctx)
	require.NoError(t, err)

	resultMap := onlyNGINXMetrics(res)
	verifyExpectations(t, resultMap, expectations)
}

func TestPrometheusIntegration_Counters(t *testing.T) {
	input := []testInput{
		{
			name: "nginx_ingress_controller_ingress_resources_total",
			desc: "Number of handled ingress resources",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "master",
					},
					value: 0,
				},
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "minion",
					},
					value: 0,
				},
				{
					labels: map[string]string{
						"class": "nginx",
						"type":  "regular",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_nginx_reload_errors_total",
			desc: "Number of unsuccessful NGINX reloads",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 0,
				},
			},
		},
		{
			name: "nginx_ingress_controller_nginx_reloads_total",
			desc: "Number of successful NGINX reloads",
			points: []testPoint{
				{
					labels: map[string]string{
						"class":  "nginx",
						"reason": "endpoints",
					},
					value: 0,
				},
				{
					labels: map[string]string{
						"class":  "nginx",
						"reason": "other",
					},
					value: 1,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_accepted",
			desc: "Accepted client connections",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 45,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_connections_handled",
			desc: "Handled client connections",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 45,
				},
			},
		},
		{
			name: "nginx_ingress_nginx_http_requests_total",
			desc: "Total http requests",
			points: []testPoint{
				{
					labels: map[string]string{
						"class": "nginx",
					},
					value: 2127,
				},
			},
		},
	}

	expectations := []model.DataEntry{
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
					Value: int64(0.0), // 0.0 is returned as an int64 by the Prometheus library.
				},
				{
					Name: "nginx_ingress_controller_ingress_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "minion",
					},
					Value: int64(0.0),
				},
				{
					Name: "nginx_ingress_controller_ingress_resources_total",
					Labels: map[string]string{
						"class": "nginx",
						"type":  "regular",
					},
					Value: int64(0.0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_nginx_reload_errors_total",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Number of unsuccessful NGINX reloads",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_nginx_reload_errors_total",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(0),
				},
			},
		},
		{
			Name:        "nginx_ingress_controller_nginx_reloads_total",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Number of successful NGINX reloads",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_nginx_reloads_total",
					Labels: map[string]string{
						"class":  "nginx",
						"reason": "endpoints",
					},
					Value: int64(0),
				},
				{
					Name: "nginx_ingress_controller_nginx_reloads_total",
					Labels: map[string]string{
						"class":  "nginx",
						"reason": "other",
					},
					Value: int64(1),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_accepted",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Accepted client connections",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_accepted",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(45),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_connections_handled",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Handled client connections",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_connections_handled",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(45),
				},
			},
		},
		{
			Name:        "nginx_ingress_nginx_http_requests_total",
			Type:        model.Counter,
			SourceType:  model.Prometheus,
			Description: "Total http requests",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_nginx_http_requests_total",
					Labels: map[string]string{
						"class": "nginx",
					},
					Value: int64(2127),
				},
			},
		},
	}

	for _, inp := range input {
		counter := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: inp.name,
			Help: inp.desc,
		}, getKeys(inp.points[0].labels))
		for _, point := range inp.points {
			counter.With(point.labels).Add(point.value)
		}
		//nolint: revive
		defer counter.Reset()
	}

	ctx := context.Background()
	s := httptest.NewServer(promhttp.Handler())
	defer s.Close()

	scraper := NewScraper([]string{s.URL})

	res, err := scraper.Produce(ctx)
	require.NoError(t, err)

	resultMap := onlyNGINXMetrics(res)
	verifyExpectations(t, resultMap, expectations)
}

// nolint: dupl
func TestPrometheusIntegration_Histograms(t *testing.T) {
	input := []testInput{
		{
			name: "nginx_ingress_controller_workqueue_queue_duration_seconds",
			desc: "How long in seconds an item stays in workqueue before being processed",
			histBuckets: []float64{
				0.1, 0.5, 1.0, 5.0, 10.0, 50.0, math.Inf(1),
			},
			histValues: map[float64]int{
				0.1:         36,
				0.5:         0,
				5.0:         0,
				10.0:        0,
				50.0:        0,
				math.Inf(1): 0,
			},
			histLabels: map[string]string{
				"class": "nginx",
				"name":  "taskQueue",
			},
		},
		{
			name: "nginx_ingress_controller_workqueue_work_duration_seconds",
			desc: "How long in seconds processing an item from workqueue takes",
			histBuckets: []float64{
				0.1, 0.5, 1.0, 5.0, 10.0, 50.0, math.Inf(1),
			},
			histValues: map[float64]int{
				0.1:         35,
				0.5:         1,
				5.0:         0,
				10.0:        0,
				50.0:        0,
				math.Inf(1): 0,
			},
			histLabels: map[string]string{
				"class": "nginx",
				"name":  "taskQueue",
			},
		},
	}

	expectations := []model.DataEntry{
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
		{
			Name:        "nginx_ingress_controller_workqueue_work_duration_seconds",
			Type:        model.Histogram,
			SourceType:  model.Prometheus,
			Description: "How long in seconds processing an item from workqueue takes",
			Values: []model.DataPoint{
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "0.1",
					},
					Value: int64(35),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "0.5",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "1",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "5",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "10",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "50",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_bucket",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
						"le":    "+Inf",
					},
					Value: int64(36),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_sum",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					Value: float64(4.000000000000002),
				},
				{
					Name: "nginx_ingress_controller_workqueue_work_duration_seconds_count",
					Labels: map[string]string{
						"class": "nginx",
						"name":  "taskQueue",
					},
					Value: int64(36),
				},
			},
		},
	}

	for _, inp := range input {
		histogram := promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    inp.name,
			Help:    inp.desc,
			Buckets: inp.histBuckets,
		}, getKeys(inp.histLabels))
		populateBuckets(histogram, inp.histValues, inp.histLabels)
		//nolint: revive
		defer histogram.Reset()
	}

	ctx := context.Background()
	s := httptest.NewServer(promhttp.Handler())
	defer s.Close()

	scraper := NewScraper([]string{s.URL})

	res, err := scraper.Produce(ctx)
	require.NoError(t, err)

	resultMap := onlyNGINXMetrics(res)
	verifyExpectations(t, resultMap, expectations)
}

func populateBuckets(hist *prometheus.HistogramVec, counts map[float64]int, labels map[string]string) {
	for bucket, count := range counts {
		for i := 0; i < count; i++ {
			hist.With(labels).Observe(bucket)
		}
	}
}

func onlyNGINXMetrics(entries []model.DataEntry) map[string]model.DataEntry {
	res := make(map[string]model.DataEntry)
	for _, e := range entries {
		if strings.Contains(e.Name, "nginx") {
			res[e.Name] = e
		}
	}

	return res
}

func getKeys(m map[string]string) []string {
	res := make([]string, 0, len(m))
	for key := range m {
		res = append(res, key)
	}

	return res
}

func verifyExpectations(t *testing.T, resultMap map[string]model.DataEntry, expectations []model.DataEntry) {
	t.Helper()
	assert.Len(t, resultMap, len(expectations))
	for _, exp := range expectations {
		t.Run(fmt.Sprintf("verify-metric->%s", exp.Name), func(tt *testing.T) {
			result, ok := resultMap[exp.Name]
			assert.True(tt, ok, fmt.Sprintf("metric [%s] should be in results", exp.Name))
			assert.Equal(tt, exp, result)
		})
	}
}
