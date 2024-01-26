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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics/prometheus"
	opsys "github.com/nginx/agent/v3/internal/model/os"
	"github.com/stretchr/testify/assert"
)

var agentConf = config.Config{
	Version: "0.1",
	Path:    "/etc/nginx-agent/",
	Metrics: &config.Metrics{
		OTelExporterTarget: "",
		ReportInterval:     5 * time.Second,
	},
}

// TODO needs mock OTel gRPC endpoint.
func TestMetrics_Init(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "")
	})

	// Create a test server using the handler
	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	testConf := config.Config{
		Version: "0.1",
		Path:    "/etc/nginx-agent/",
		Metrics: &config.Metrics{
			OTelExporterTarget: "dummy-target",
			ReportInterval:     15 * time.Second,
		},
	}

	err := os.Setenv("PROMETHEUS_TARGETS", fakePrometheus.URL)
	if err != nil {
		log.Fatalf("err %v", err)
	}

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	scraper := prometheus.NewScraper()

	metrics, err := NewMetrics(testConf, WithDataSource(scraper))
	assert.NoError(t, err)

	err = messagePipe.Register(100, []bus.Plugin{metrics})
	assert.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	assert.NoError(t, err)

	err = metrics.Close()
	assert.NoError(t, err)
}

func TestMetrics_Info(t *testing.T) {
	metrics, err := NewMetrics(agentConf)
	assert.NoError(t, err)

	i := metrics.Info()
	assert.NotNil(t, i)

	assert.Equal(t, "metrics", i.Name)
}

func TestMetrics_Subscriptions(t *testing.T) {
	metrics, err := NewMetrics(agentConf)
	assert.NoError(t, err)

	subscriptions := metrics.Subscriptions()
	assert.Equal(t, []string{bus.OS_PROCESSES_TOPIC}, subscriptions)
}

func TestMetrics_Process(t *testing.T) {
	metrics, err := NewMetrics(agentConf)
	assert.NoError(t, err)

	metrics.Process(&bus.Message{Topic: bus.OS_PROCESSES_TOPIC, Data: []*opsys.Process{{Pid: 123, Name: "nginx"}}})

	// Currently doesn't do anything.
	assert.NoError(t, err)
}
