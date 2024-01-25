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

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/metrics"
)

const (
	defaultTarget  = "http://127.0.0.1:30882/metrics"
	DataSourceType = "PROMETHEUS"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_message_bus.go . MessageBusContract
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/metrics/prometheus mock_message_bus.go | sed -e s\\/prometheus\\\\.\\/\\/g > mock_message_bus_fixed.go"
//go:generate mv mock_message_bus_fixed.go mock_message_bus.go
type MessageBusContract interface {
	Register(int, []bus.Plugin) error
	DeRegister(plugins []string) error
	Process(...*bus.Message)
	Run()
	Context() context.Context
	GetPlugins() []bus.Plugin
	IsPluginAlreadyRegistered(string) bool
}

// Used for deserializion of parsed Prometheus data.
type Scraper struct {
	bus bus.MessagePipeInterface
	// previousCounterMetricValues map[string]DataPoint
	cancelFunc context.CancelFunc
}

func NewScraper(mp bus.MessagePipeInterface) *Scraper {
	return &Scraper{
		bus: mp,
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
				msgs := []*bus.Message{}
				for _, e := range entries {
					msgs = append(msgs, e.ToBusMessage())
				}
				s.bus.Process(msgs...)
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

func scrapeEndpoint(target string) []metrics.DataEntry {
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

	entries := make([]metrics.DataEntry, 0)

	currEntry := new(metrics.DataEntry)
	currEntry.SourceType = DataSourceType
	for _, line := range splitResponse {
		if strings.HasPrefix(line, "# HELP") {
			// If entry already has content, then we add it to the result slice and reset.
			if currEntry.Name != "" {
				entries = append(entries, *currEntry)
				currEntry = new(metrics.DataEntry)
				currEntry.SourceType = DataSourceType
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

			pointName := splitNameAndLabels[0]
			var pointErr error
			if len(splitNameAndLabels) > 1 {
				labelsJson := strings.ReplaceAll("{\""+splitNameAndLabels[1], "=", "\":")
				labelsJson = strings.ReplaceAll(labelsJson, ",", ",\"")
				err := json.Unmarshal([]byte(labelsJson), &labels)
				if err != nil {
					pointErr = errors.Join(pointErr, fmt.Errorf("failed to parse Prometheus data point labels [%s]: %w", splitNameAndLabels, err))
				}
			}

			value, err := strconv.ParseFloat(splitMetric[1], 64)
			if err != nil {
				pointErr = errors.Join(pointErr, fmt.Errorf("failed to parse Prometheus data point value [%s]: %w", splitMetric[1], err))
			}

			if pointErr == nil {
				currEntry.Values = append(currEntry.Values, metrics.DataPoint{
					Name:   pointName,
					Labels: labels,
					Value:  value,
				})
			} else {
				// Discard a data point that has malformed content, so we don't present dirty data to end users.
				slog.Warn("Prometheus data point parsing error, discarding point", "errors", pointErr)
			}

		}
	}

	slog.Debug("Prometheus endpoint scraped", "number of data entries", len(entries), "Prometheus URL", target)
	return entries
}
