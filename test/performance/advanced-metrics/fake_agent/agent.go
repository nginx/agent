package fake_agent

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/schema"
)

const (
	aggregatedValue              = "AGGR"
	advancedMetricsPluginVersion = "v0.8.0"
	advancedMetricsPluginName    = "Advanced Metrics Plugin"

	httpUriDimension                   = "http.uri"
	httpResponseCodeDimension          = "http.response_code"
	httpRequestMethodDimension         = "http.request_method"
	hitcountMetric                     = "hitcount"
	httpRequestBytesRcvdMetric         = "http.request.bytes_rcvd"
	httpRequestBytesSentMetric         = "http.request.bytes_sent"
	environmentDimension               = "environment"
	appDimension                       = "app"
	componentDimension                 = "component"
	countryCodeDimension               = "country_code"
	httpVersionSchemaDimension         = "http.version_schema"
	httpUpstreamAddrDimension          = "http.upstream_addr" // TODO this should not contain http. prefix probably
	upstreamResponseCodeDimension      = "upstream_response_code"
	httpHostnameDimension              = "http.hostname"
	clientNetworkLatencyMetric         = "client.network.latency"
	clientTtfbLatencyMetric            = "client.ttfb.latency"
	clientRequestLatencyMetric         = "client.request.latency"
	clientResponseLatencyMetric        = "client.response.latency"
	upstreamNetworkLatencyMetric       = "upstream.network.latency"
	upstreamHeaderLatencyMetric        = "upstream.header.latency"
	upstreamResponseLatencyMetric      = "upstream.response.latency"
	publishedApiDimension              = "published_api"
	requestOutcomeDimension            = "request_outcome"
	requestOutcomeReasonDimension      = "request_outcome_reason"
	gatewayDimension                   = "gateway"
	wafSignatureIdsDimension           = "waf.signature_ids"
	wafAttackTypesDimension            = "waf.attack_types"
	wafViolationRatingDimension        = "waf.violation_rating"
	wafViolationsDimension             = "waf.violations"
	wafViolationSubviolationsDimension = "waf.violation_subviolations"
	clientLatencyMetric                = "client.latency"
	upstreamLatencyMetric              = "upstream.latency"
	connectionDurationMetric           = "connection_duration"
	familyDimension                    = "family"
	proxiedProtocolDimension           = "proxied_protocol"
	bytesRcvdMetric                    = "bytes_rcvd"
	bytesSentMetric                    = "bytes_sent"

	aggregationDurrationDimension = "aggregation_duration"

	streamMetricFamilyDimensionValue = "tcp-udp"
)

var (
	messagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_processed_total",
		Help: "The total number of processed messages",
	})
	metricsProcessedOnOutput = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_processed_dimension_set",
		Help: "The total number of processed entities",
	})
	aggregatedDimensionValuesProcessedOnOutput = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_discovered_aggregated_dimension_values",
		Help: "Number of processed dimensions with AGGR value",
	})
)

type Config struct {
	AdvancedMetricsSocket string `envconfig:"advanced_metrics_socket" default:"/tmp/bench.sock"`
	PromPort              int    `envconfig:"prometheus_port" default:"2112"`
}

func main() {
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		log.Fatalf("cannot process configuration: %s", err.Error())
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		fmt.Println("exposing metrics on port", cfg.PromPort)
		mux.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(fmt.Sprintf(":%v", cfg.PromPort), mux))
	}()

	builder := schema.NewSchemaBuilder()
	builder.NewDimension(httpUriDimension, 16000).
		NewIntegerDimension(httpResponseCodeDimension, 600).
		NewDimension(httpRequestMethodDimension, 16).
		NewMetric(hitcountMetric).
		NewMetric(httpRequestBytesRcvdMetric).
		NewMetric(httpRequestBytesSentMetric).
		NewDimension(environmentDimension, 32).
		NewDimension(appDimension, 32).
		NewDimension(componentDimension, 256).
		NewDimension(countryCodeDimension, 256). //TODO should be implemented as GeoIP
		NewDimension(httpVersionSchemaDimension, 16).
		NewDimension(httpUpstreamAddrDimension, 1024).
		NewIntegerDimension(upstreamResponseCodeDimension, 600).
		NewDimension(httpHostnameDimension, 16000).
		NewMetric(clientNetworkLatencyMetric).
		NewMetric(clientTtfbLatencyMetric).
		NewMetric(clientRequestLatencyMetric).
		NewMetric(clientResponseLatencyMetric).
		NewMetric(upstreamNetworkLatencyMetric).
		NewMetric(upstreamHeaderLatencyMetric).
		NewMetric(upstreamResponseLatencyMetric).
		NewDimension(publishedApiDimension, 256).
		NewDimension(requestOutcomeDimension, 8).
		NewDimension(requestOutcomeReasonDimension, 32).
		NewDimension(gatewayDimension, 32).
		NewDimension(wafSignatureIdsDimension, 16000).
		NewDimension(wafAttackTypesDimension, 8).
		NewDimension(wafViolationRatingDimension, 8).
		NewDimension(wafViolationsDimension, 128).
		NewDimension(wafViolationSubviolationsDimension, 16).
		NewMetric(clientLatencyMetric).
		NewMetric(upstreamLatencyMetric).
		NewMetric(connectionDurationMetric).
		NewDimension(familyDimension, 4).
		NewDimension(proxiedProtocolDimension, 4).
		NewMetric(bytesRcvdMetric).
		NewMetric(bytesSentMetric)

	config := advanced_metrics.Config{
		Address: cfg.AdvancedMetricsSocket,
	}
	config.AggregationPeriod = time.Second * 10
	config.PublishingPeriod = time.Second * 30
	config.StagingTableMaxSize = 32000
	config.StagingTableThreshold = 28000
	schema, err := builder.Build()
	if err != nil {
		log.Fatal(err)
	}
	app, err := advanced_metrics.NewAdvancedMetrics(config, schema)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for f := range app.OutChannel() {
			for _, m := range f {
				messagesProcessed.Inc()
				metricsProcessedOnOutput.Add(float64(len(m.Metrics)))
			}
			if len(f) == 0 {
				continue
			}
			for _, out := range f {
				for _, d := range out.Dimensions {
					if d.Value == aggregatedValue {
						aggregatedDimensionValuesProcessedOnOutput.Inc()
					}
				}
			}
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())

	interuptSignal := make(chan os.Signal, 1)
	signal.Notify(interuptSignal, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-interuptSignal
		logrus.Info("Stopping fake agent")
		cancel()
	}()

	err = app.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
