package integration

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/schema"
	conf "github.com/nginx/agent/v2/test/integration/nginx"
	"github.com/nginx/agent/v2/test/integration/upstream"
	"github.com/nginx/agent/v2/test/integration/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SetupAdvancedMetrics(t *testing.T, socketLocation string) *advanced_metrics.AdvancedMetrics {
	builder := schema.NewSchemaBuilder()
	builder.NewDimension("http.uri", 16000).
		NewIntegerDimension("http.response_code", 600).
		NewDimension("http.request_method", 16).
		NewMetric("http.request.count").
		NewMetric("http.request.bytes_rcvd").
		NewMetric("http.request.bytes_sent").
		NewDimension("environment", 32).
		NewDimension("app", 32).
		NewDimension("component", 256).
		NewDimension("country_code", 256).
		NewDimension("http.version_schema", 16).
		NewDimension("http.upstream_addr", 1024).
		NewIntegerDimension("upstream_response_code", 600).
		NewDimension("http.hostname", 16000).
		NewMetric("client.network.latency").
		NewMetric("client.ttfb.latency").
		NewMetric("client.request.latency").
		NewMetric("client.response.latency").
		NewMetric("upstream.network.latency").
		NewMetric("upstream.header.latency").
		NewMetric("upstream.response.latency").
		NewDimension("published_api", 256).
		NewDimension("request_outcome", 8).
		NewDimension("request_outcome_reason", 32).
		NewDimension("gateway", 32).
		NewDimension("waf.signature_ids", 16000).
		NewDimension("waf.attack_types", 8).
		NewDimension("waf.violation_rating", 8).
		NewDimension("waf.violations", 128).
		NewDimension("waf.violation_subviolations", 16).
		NewMetric("client.latency").
		NewMetric("upstream.latency").
		NewMetric("connection_duration").
		NewDimension("family", 4).
		NewDimension("proxied_protocol", 4).
		NewMetric("bytes_rcvd").
		NewMetric("bytes_sent")

	cfg := advanced_metrics.Config{
		Address: socketLocation,
		AggregatorConfig: advanced_metrics.AggregatorConfig{
			AggregationPeriod: time.Second,
			PublishingPeriod:  time.Second * 10,
		},
		TableSizesLimits: advanced_metrics.TableSizesLimits{
			StagingTableMaxSize:    1000,
			StagingTableThreshold:  1000,
			PriorityTableMaxSize:   1000,
			PriorityTableThreshold: 1000,
		},
	}

	s, err := builder.Build()
	assert.NoError(t, err)
	r, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := r.Run(ctx)
		assert.NoError(t, err)
	}()

	assert.Eventually(t, func() bool {
		err := os.Chmod(socketLocation, 0666)
		return err == nil
	}, time.Second*2, time.Microsecond*100, "fail to change socket file permission")

	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	return r
}

const (
	location1         = "/loc1"
	upstreamHost      = "localhost"
	upstreamAddress   = "127.0.0.1"
	upstreamPort      = 9090
	httpServerAddress = "127.0.0.2"
	tcpServerAddress  = "127.0.0.3"

	env1  = "env1"
	app1  = "app1"
	gw1   = "gw1"
	comp1 = "component1"
)

func TestBasicHttpRequestMetrics(t *testing.T) {
	httpServerListenAddress := fmt.Sprintf("%s:8080", httpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	httpUpstream := upstream.HttpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort),
		Handlers: map[string]upstream.Handler{
			location1: {
				Handler: func(w http.ResponseWriter, req *http.Request) {
					fmt.Fprintf(w, "OK")
				},
			},
		},
	}
	httpUpstream.Serve(t)

	cfg := conf.NginxConf{
		HttpBlock: &conf.HttpBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				httpUpstream.Name: httpUpstream.AsUpstream(),
			},
			Servers: []conf.Server{
				httpUpstream.AsServer(httpServerListenAddress,
					map[string]string{
						conf.MarkerGateway: gw1,
					},
					upstream.LocationsMarkers{
						location1: map[string]string{
							conf.MarkerApp:       app1,
							conf.MarkerComponent: comp1,
						},
					},
					upstream.LocationsDirectives{},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	location1Url := fmt.Sprintf("http://%s/loc1", httpServerListenAddress)

	_, err = http.Get(location1Url)
	assert.NoError(t, err)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "http.uri", Value: location1},
		{Name: "http.request_method", Value: "GET"},
		{Name: "http.response_code", Value: "200"},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.version_schema", Value: "4"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "http.hostname", Value: httpServerAddress},
		{Name: "family", Value: "web"},
		{Name: "proxied_protocol", Value: "http"},
		{Name: "request_outcome", Value: "PASSED"},
		{Name: "upstream_response_code", Value: "200"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2, High: 2},
		},
		{
			Name:     "http.request.bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "http.request.bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
		{
			Name:     "client.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.ttfb.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.request.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.header.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestOutcomRejectedWithWafMetrics(t *testing.T) {
	httpServerListenAddress := fmt.Sprintf("%s:8080", httpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	httpUpstream := upstream.HttpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort),
		Handlers: map[string]upstream.Handler{
			location1: {
				Handler: func(w http.ResponseWriter, req *http.Request) {
					fmt.Fprintf(w, "OK")
				},
			},
		},
	}
	httpUpstream.Serve(t)

	cfg := conf.NginxConf{
		HttpBlock: &conf.HttpBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				httpUpstream.Name: httpUpstream.AsUpstream(),
			},
			Servers: []conf.Server{
				httpUpstream.AsServer(httpServerListenAddress,
					map[string]string{
						conf.MarkerGateway: gw1,
					},
					upstream.LocationsMarkers{
						location1: map[string]string{
							conf.MarkerApp:       app1,
							conf.MarkerComponent: comp1,
						},
					},
					upstream.LocationsDirectives{
						location1: []string{
							"set $app_protect_outcome REJECTED",
							"set $app_protect_signature_ids \"123456789,987654321\"",
							"set $app_protect_attack_types \"mock_attack_types\"",
							"set $app_protect_violation_rating \"likely_attack\"",
							"set $app_protect_violations \"Something with Space\"",
							"set $app_protect_violation_subviolations \"Empty\"",
						},
					},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	location1Url := fmt.Sprintf("http://%s/loc1", httpServerListenAddress)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "http.uri", Value: location1},
		{Name: "http.request_method", Value: "GET"},
		{Name: "http.response_code", Value: "200"},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.version_schema", Value: "4"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "http.hostname", Value: httpServerAddress},
		{Name: "family", Value: "web"},
		{Name: "proxied_protocol", Value: "http"},
		{Name: "request_outcome", Value: "REJECTED"},
		{Name: "upstream_response_code", Value: "200"},
		{Name: "waf.attack_types", Value: "mock_attack_types"},
		{Name: "waf.signature_ids", Value: "123456789,987654321"},
		{Name: "waf.violation_rating", Value: "likely_attack"},
		{Name: "waf.violations", Value: "Something with Space"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2, High: 2},
		},
		{
			Name:     "http.request.bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "http.request.bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
		{
			Name:     "client.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.ttfb.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.request.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.header.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestUrlWithStringFieldEscapeCharacter(t *testing.T) {
	t.Skip()

	locationWithQuote := location1 + ` " `
	httpServerListenAddress := fmt.Sprintf("%s:8080", httpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	httpUpstream := upstream.HttpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort),
		Handlers: map[string]upstream.Handler{
			locationWithQuote: {
				Handler: func(w http.ResponseWriter, req *http.Request) {
					fmt.Fprintf(w, "OK")
				},
			},
		},
	}
	httpUpstream.Serve(t)

	cfg := conf.NginxConf{
		HttpBlock: &conf.HttpBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				httpUpstream.Name: httpUpstream.AsUpstream(),
			},
			Servers: []conf.Server{
				httpUpstream.AsServer(httpServerListenAddress,
					map[string]string{
						conf.MarkerGateway: gw1,
					},
					upstream.LocationsMarkers{
						locationWithQuote: map[string]string{
							conf.MarkerApp:       app1,
							conf.MarkerComponent: comp1,
						},
					},
					upstream.LocationsDirectives{},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	location1Url := fmt.Sprintf("http://%s/%s", httpServerListenAddress, locationWithQuote)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "http.uri", Value: locationWithQuote},
		{Name: "http.request_method", Value: "GET"},
		{Name: "http.response_code", Value: "200"},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.version_schema", Value: "4"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "http.hostname", Value: httpServerAddress},
		{Name: "family", Value: "web"},
		{Name: "proxied_protocol", Value: "http"},
		{Name: "request_outcome", Value: "PASSED"},
		{Name: "upstream_response_code", Value: "200"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2, High: 2},
		},
		{
			Name:     "http.request.bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "http.request.bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
		{
			Name:     "client.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.ttfb.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.request.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.header.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestHttpRequestMetricsWithNonZeroLatency(t *testing.T) {
	httpServerListenAddress := fmt.Sprintf("%s:8080", httpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	httpUpstream := upstream.HttpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort),
		Handlers: map[string]upstream.Handler{
			location1: {
				Handler: func(w http.ResponseWriter, req *http.Request) {
					time.Sleep(100 * time.Millisecond)
					fmt.Fprintf(w, "OK")
				},
			},
		},
	}
	httpUpstream.Serve(t)

	cfg := conf.NginxConf{
		HttpBlock: &conf.HttpBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				httpUpstream.Name: httpUpstream.AsUpstream(),
			},
			Servers: []conf.Server{
				httpUpstream.AsServer(httpServerListenAddress,
					map[string]string{
						conf.MarkerGateway: gw1,
					},
					upstream.LocationsMarkers{
						location1: map[string]string{
							conf.MarkerApp:       app1,
							conf.MarkerComponent: comp1,
						},
					},
					upstream.LocationsDirectives{},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	location1Url := fmt.Sprintf("http://%s/loc1", httpServerListenAddress)

	_, err = http.Get(location1Url)
	assert.NoError(t, err)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "http.uri", Value: location1},
		{Name: "http.request_method", Value: "GET"},
		{Name: "http.response_code", Value: "200"},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.version_schema", Value: "4"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "http.hostname", Value: httpServerAddress},
		{Name: "family", Value: "web"},
		{Name: "proxied_protocol", Value: "http"},
		{Name: "request_outcome", Value: "PASSED"},
		{Name: "upstream_response_code", Value: "200"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2, High: 2},
		},
		{
			Name:     "http.request.bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "http.request.bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
		{
			Name:     "client.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.ttfb.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "client.request.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.response.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "upstream.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.header.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "upstream.response.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "client.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "upstream.latency",
			MinRange: validator.Range{Low: 100, High: 102}, MaxRange: validator.Range{Low: 100, High: 102},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 200, High: 204},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: 164, High: 164}, MaxRange: validator.Range{Low: 164, High: 164},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 164, High: 2 * 164},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestHttpRequestMetricsWith5kResponseSize(t *testing.T) {
	httpServerListenAddress := fmt.Sprintf("%s:8080", httpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	httpUpstream := upstream.HttpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamHost, upstreamPort),
		Handlers: map[string]upstream.Handler{
			location1: {
				Handler: func(w http.ResponseWriter, req *http.Request) {
					responseLen := 5 * 1024
					bytes := make([]byte, responseLen)
					for i := 0; i < responseLen; i++ {
						bytes[i] = byte(rand.Intn(100))
					}
					fmt.Fprint(w, string(bytes))
				},
			},
		},
	}
	httpUpstream.Serve(t)

	cfg := conf.NginxConf{
		HttpBlock: &conf.HttpBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				httpUpstream.Name: httpUpstream.AsUpstream(),
			},
			Servers: []conf.Server{
				httpUpstream.AsServer(httpServerListenAddress,
					map[string]string{
						conf.MarkerGateway: gw1,
					},
					upstream.LocationsMarkers{
						location1: map[string]string{
							conf.MarkerApp:       app1,
							conf.MarkerComponent: comp1,
						},
					},
					upstream.LocationsDirectives{},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	location1Url := fmt.Sprintf("http://%s/loc1", httpServerListenAddress)

	_, err = http.Get(location1Url)
	assert.NoError(t, err)
	_, err = http.Get(location1Url)
	assert.NoError(t, err)

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "http.uri", Value: location1},
		{Name: "http.request_method", Value: "GET"},
		{Name: "http.response_code", Value: "200"},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.version_schema", Value: "4"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "http.hostname", Value: httpServerAddress},
		{Name: "family", Value: "web"},
		{Name: "proxied_protocol", Value: "http"},
		{Name: "request_outcome", Value: "PASSED"},
		{Name: "upstream_response_code", Value: "200"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2, High: 2},
		},
		{
			Name:     "http.request.bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "http.request.bytes_sent",
			MinRange: validator.Range{Low: 5309 - 100, High: 5309 + 100}, MaxRange: validator.Range{Low: 5309 - 100, High: 5309 + 100},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 5309*2 - 200, High: 5309*2 + 200},
		},
		{
			Name:     "client.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.ttfb.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.request.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.network.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.header.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.response.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "client.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "upstream.latency",
			MinRange: validator.Range{Low: 0, High: 2}, MaxRange: validator.Range{Low: 0, High: 2},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 0, High: 4},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: 99, High: 99}, MaxRange: validator.Range{Low: 99, High: 99},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 2 * 99, High: 2 * 99},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: 5309 - 100, High: 5309 + 100}, MaxRange: validator.Range{Low: 5309 - 100, High: 5309 + 100},
			CountRange: validator.Range{Low: 2, High: 2}, SumRange: validator.Range{Low: 5309*2 - 200, High: 5309*2 + 200},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestStreamTcpMetrics(t *testing.T) {
	tcpServerListenAddress := fmt.Sprintf("%s:8080", tcpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	tcpUpstream := upstream.TcpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort),
		Handler: func(conn net.Conn) {
			_, err := io.Copy(conn, conn)
			assert.NoError(t, err)
		},
	}
	tcpUpstream.Serve(t)

	cfg := conf.NginxConf{
		StreamBlock: &conf.StreamBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				tcpUpstream.Name: tcpUpstream.AsUpstream(),
			},
			Servers: []conf.StreamServer{
				tcpUpstream.AsServer(tcpServerListenAddress,
					map[string]string{
						conf.MarkerGateway:   gw1,
						conf.MarkerApp:       app1,
						conf.MarkerComponent: comp1,
					},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	testData := []byte("test_data_test_data")
	testDataLength := len(testData)
	conn, err := net.Dial("tcp", tcpServerListenAddress)
	assert.NoError(t, err)
	send, err := conn.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, send, len(testData))

	buff := make([]byte, 200)
	received, err := conn.Read(buff)
	assert.Equal(t, received, len(testData))
	assert.NoError(t, err)

	assert.Equal(t, buff[:received], testData)

	const connectionDurationSeconds = 2
	const connectionDurationMiliseconds = 2 * 1000
	time.Sleep(time.Second * connectionDurationSeconds)
	conn.Close()

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "family", Value: "tcp-udp"},
		{Name: "proxied_protocol", Value: "tcp"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: 1, High: 1},
		},
		{
			Name:     "connection_duration",
			MinRange: validator.Range{Low: connectionDurationMiliseconds, High: connectionDurationMiliseconds + 2}, MaxRange: validator.Range{Low: connectionDurationMiliseconds, High: connectionDurationMiliseconds + 2},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: connectionDurationMiliseconds, High: connectionDurationMiliseconds + 2},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)}, MaxRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)}, MaxRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}

func TestStreamUdpMetrics(t *testing.T) {
	tcpServerListenAddress := fmt.Sprintf("%s:8080", tcpServerAddress)

	socket := "/tmp/advanced_metrics.sr"
	advanced_metrics := SetupAdvancedMetrics(t, socket)

	udpUpstream := upstream.UdpTestUpstream{
		Name:    "upstream1",
		Address: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort),
		Handler: func(pc net.PacketConn, a net.Addr, b []byte) {
			send, err := pc.WriteTo(b, a)
			assert.NoError(t, err)
			assert.Equal(t, len(b), send)
		},
	}
	udpUpstream.Serve(t)

	connectionTimeout := time.Second * 5

	cfg := conf.NginxConf{
		StreamBlock: &conf.StreamBlock{
			F5MetricsServer: socket,
			F5MetricsMarkers: map[string]string{
				conf.MarkerEnvironment: env1,
			},
			Upstreams: map[string]conf.Upstream{
				udpUpstream.Name: udpUpstream.AsUpstream(),
			},
			Servers: []conf.StreamServer{
				udpUpstream.AsServer(tcpServerListenAddress,
					map[string]string{
						conf.MarkerGateway:   gw1,
						conf.MarkerApp:       app1,
						conf.MarkerComponent: comp1,
					},
					[]string{
						fmt.Sprintf("proxy_timeout %s", connectionTimeout.String()),
					},
				),
			},
		},
	}
	cmd, err := conf.NewNginxCommand(&cfg)
	require.NoError(t, err)
	cmd.Start(t)

	testData := []byte("test_data_test_data")
	testDataLength := len(testData)
	addr, err := net.ResolveUDPAddr("udp", tcpServerListenAddress)
	assert.NoError(t, err)
	conn, err := net.DialUDP("udp", nil, addr)
	assert.NoError(t, err)
	send, err := conn.Write(testData)
	assert.NoError(t, err)
	assert.Equal(t, send, len(testData))

	buff := make([]byte, 200)
	received, _, err := conn.ReadFromUDP(buff)

	assert.Equal(t, received, len(testData))
	assert.NoError(t, err)

	assert.Equal(t, buff[:received], testData)

	conn.Close()

	expectedDimensions := []publisher.Dimension{
		{Name: "app", Value: app1},
		{Name: "component", Value: comp1},
		{Name: "gateway", Value: gw1},
		{Name: "environment", Value: env1},
		{Name: "country_code", Value: "0100007fffff00000000000000000000"},
		{Name: "http.upstream_addr", Value: fmt.Sprintf("%s:%d", upstreamAddress, upstreamPort)},
		{Name: "family", Value: "tcp-udp"},
		{Name: "proxied_protocol", Value: "udp"},
	}

	expectedMetrics := []validator.ExpectedMetric{
		{
			Name:     "http.request.count",
			MinRange: validator.Range{Low: 1, High: 1}, MaxRange: validator.Range{Low: 1, High: 1},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: 1, High: 1},
		},
		{
			Name:     "connection_duration",
			MinRange: validator.Range{Low: float64(connectionTimeout.Milliseconds()) - 3, High: float64(connectionTimeout.Milliseconds()) + 3}, MaxRange: validator.Range{Low: float64(connectionTimeout.Milliseconds()) - 3, High: float64(connectionTimeout.Milliseconds()) + 2},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: float64(connectionTimeout.Milliseconds()) - 3, High: float64(connectionTimeout.Milliseconds()) + 2},
		},
		{
			Name:     "bytes_rcvd",
			MinRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)}, MaxRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
		},
		{
			Name:     "bytes_sent",
			MinRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)}, MaxRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
			CountRange: validator.Range{Low: 1, High: 1}, SumRange: validator.Range{Low: float64(testDataLength), High: float64(testDataLength)},
		},
	}

	select {
	case metrics := <-advanced_metrics.OutChannel():
		assert.NotEmpty(t, metrics)
		assert.Len(t, metrics, 1)
		validator.AssertMetricSetEqual(t, expectedMetrics, expectedDimensions, metrics[0])
	case <-time.After(time.Second * 15):
		assert.Fail(t, "failed to receive message")
	}
}
