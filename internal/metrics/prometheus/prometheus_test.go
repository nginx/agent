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

	"github.com/stretchr/testify/assert"
)

const (
	prometheusTestDataFile = "prometheus_test_data.txt"
	agentVersion           = "0.1"
)

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
	scraper := NewScraper()
	assert.NotNil(t, scraper)
}

func TestPrometheus_Start(t *testing.T) {
	scraper := NewScraper()
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
	scraper := NewScraper()
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
	scraper := NewScraper()
	assert.Equal(t, "PROMETHEUS", scraper.Type())
}
