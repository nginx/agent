// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nginx/agent/v3/internal/model"
)

const (
	prometheusEnvVar = "PROMETHEUS_TARGETS"
	// Prefixes that label the content of Prometheus data rows.
	prometheusHelpPrefix = "# HELP"
	prometheusTypePrefix = "# TYPE"

	helpLineSpaces  = 4
	typeLineSpaces  = 4
	pointLineSpaces = 3
)

// Scraper is a Prometheus MetricsProducer.
type Scraper struct {
	endpoints  []string
	cancelFunc context.CancelFunc
}

// NewScraper initializes a new Prometheus MetricsProducer.
func NewScraper(targets []string) *Scraper {
	if len(targets) == 0 {
		if value, ok := os.LookupEnv(prometheusEnvVar); ok {
			targets = strings.Split(value, ",")
		}
	}

	return &Scraper{
		endpoints:  targets,
		cancelFunc: nil,
	}
}

func (s *Scraper) Produce(ctx context.Context) ([]model.DataEntry, error) {
	slog.Debug("Prometheus collection called")

	cancellableCtx, cancelFunc := context.WithCancel(ctx)
	s.cancelFunc = cancelFunc

	mut := sync.Mutex{}
	wg := &sync.WaitGroup{}
	errChan := make(chan error)

	results := make([]model.DataEntry, 0)
	// Query each endpoint in its own goroutine.
	for _, ep := range s.endpoints {
		wg.Add(1)
		go func(target string, mu *sync.Mutex) {
			defer wg.Done()
			entries, err := scrapeEndpoint(cancellableCtx, strings.TrimSpace(target))
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			results = append(results, entries...)
			mu.Unlock()
		}(ep, &mut)
	}

	wg.Wait()
	close(errChan)

	var endpointErrs error
	for incErr := range errChan {
		endpointErrs = errors.Join(endpointErrs, incErr)
	}

	if endpointErrs != nil {
		return results, fmt.Errorf("prometheus scrape error(s): %w", endpointErrs)
	}

	return results, nil
}

func (s *Scraper) Type() model.MetricsSourceType {
	return model.Prometheus
}

// SplitN arguments are dependent on what Prometheus data looks like for each type of line.
func scrapeEndpoint(ctx context.Context, target string) ([]model.DataEntry, error) {
	rows, err := fetchPrometheusData(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus metrics endpoint: %w", err)
	}

	entries := make([]model.DataEntry, 0)
	builder := model.NewEntryBuilder(model.WithSourceType(model.Prometheus))
	for _, row := range rows {
		entries = processRow(row, &builder, entries)
	}

	// Add the final entry being parsed, as the loop above adds entries only on '# HELP' lines, and the final line is
	// NOT a '# HELP' line
	if builder.CanBuild() {
		entries = append(entries, builder.Build())
	}

	slog.Debug("Prometheus endpoint scraped", "data_entry_count", len(entries), "prometheus_url", target)

	return entries, nil
}

func fetchPrometheusData(ctx context.Context, targetURL string) ([]string, error) {
	responseString, err := queryTarget(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	return strings.Split(responseString, "\n"), nil
}

// queries the given URL and returns its content as a string.
func queryTarget(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to formulate Prometheus HTTP scrape endpoint GET request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET request to Prometheus HTTP scrape endpoint [%s] failed: %w", url, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse Prometheus scrape endpoint response body: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close response body: %w", err)
	}

	return string(body), nil
}

func processRow(row string, builder *model.EntryBuilder, entries []model.DataEntry) []model.DataEntry {
	result := entries
	if strings.HasPrefix(row, prometheusHelpPrefix) {
		result = appendEntry(builder, entries)
		processHelpRow(row, builder)
	} else if strings.HasPrefix(row, prometheusTypePrefix) {
		processTypeRow(row, builder)
	} else if row != "" {
		processPointRow(row, builder)
	}

	return result
}

// Appends the contents of the builder into the result slice if builder is non-empty.
func appendEntry(builder *model.EntryBuilder, entries []model.DataEntry) []model.DataEntry {
	result := entries
	// If entry already has content, then we add it to the result slice and reset.
	if builder.CanBuild() {
		result = append(result, builder.Build())
		*builder = model.NewEntryBuilder(model.WithSourceType(model.Prometheus))
	}

	return result
}

func processHelpRow(row string, builder *model.EntryBuilder) {
	splitHelp := strings.SplitN(row, " ", helpLineSpaces)

	builder.WithName(splitHelp[2])
	builder.WithDescription(splitHelp[3])
}

func processTypeRow(row string, builder *model.EntryBuilder) {
	splitType := strings.SplitN(row, " ", typeLineSpaces)
	builder.WithType(model.ToInstrumentType(splitType[3]))
}

func processPointRow(row string, builder *model.EntryBuilder) {
	splitMetric := strings.SplitN(row, " ", pointLineSpaces)
	if len(splitMetric) <= 1 {
		// Line has no value so we skip.
		return
	}
	nameAndLabels := splitMetric[0]
	splitNameAndLabels := strings.Split(nameAndLabels, "{")

	pointName := splitNameAndLabels[0]
	var (
		pointErr error
		labels   map[string]string
	)
	if len(splitNameAndLabels) > 1 {
		labels, pointErr = parseLabels(splitNameAndLabels[1])
	}

	value, err := parseValue(splitMetric[1])
	if err != nil {
		pointErr = errors.Join(pointErr, err)
	}

	if pointErr == nil {
		builder.WithValues(model.DataPoint{
			Name:   pointName,
			Labels: labels,
			Value:  value,
		})
	} else {
		// Discard a data point that has malformed content, so we don't present dirty data to end users.
		slog.Debug("Prometheus data point parsing error: discarding point", "error", pointErr)
	}
}

// Parses a Prometheus data point's labels. An example data point:
// {dialer_name="prometheus",reason="unknown"}
func parseLabels(labelsInput string) (map[string]string, error) {
	labels := make(map[string]string)
	labelsJSON := strings.ReplaceAll("{\""+labelsInput, "=", "\":")
	labelsJSON = strings.ReplaceAll(labelsJSON, ",", ",\"")
	err := json.Unmarshal([]byte(labelsJSON), &labels)
	if err != nil {
		return labels, fmt.Errorf("failed to parse Prometheus data point labels [%s]: %w", labelsInput, err)
	}

	return labels, nil
}

// parseValue coerces the string value of a number to either float or int.
func parseValue(input string) (any, error) {
	var (
		result any
		err    error
	)

	if strings.Contains(input, ".") {
		result, err = strconv.ParseFloat(input, 64)
	} else {
		result, err = strconv.ParseInt(input, 10, 64)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to parse Prometheus data point value [%s]: %w", input, err)
	}

	return result, nil
}
