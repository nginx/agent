package plugins

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/schema"
	log "github.com/sirupsen/logrus"
)

const (
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

	aggregationDurationDimension = "aggregation_duration"

	streamMetricFamilyDimensionValue = "tcp-udp"
)

var maxOnlyMetrics = map[string]struct{}{
	clientNetworkLatencyMetric:    {},
	clientTtfbLatencyMetric:       {},
	clientRequestLatencyMetric:    {},
	clientResponseLatencyMetric:   {},
	upstreamNetworkLatencyMetric:  {},
	upstreamHeaderLatencyMetric:   {},
	upstreamResponseLatencyMetric: {},
}

var totalOnlyMetrics = map[string]struct{}{
	bytesRcvdMetric:          {},
	bytesSentMetric:          {},
	connectionDurationMetric: {},
}

const httpMetricPrefix = "http.request"
const streamMetricPrefix = "stream"

type AdvancedMetrics struct {
	ctx              context.Context
	ctxCancel        context.CancelFunc
	cfg              advanced_metrics.Config
	advanced_metrics *advanced_metrics.AdvancedMetrics
	pipeline         core.MessagePipeInterface
	commonDims       *metrics.CommonDim
}

func NewAdvancedMetrics(env core.Environment, conf *config.Config) *AdvancedMetrics {
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

	cfg := advanced_metrics.Config{
		Address: conf.AdvancedMetrics.SocketPath,
		AggregatorConfig: advanced_metrics.AggregatorConfig{
			AggregationPeriod: conf.AdvancedMetrics.AggregationPeriod,
			PublishingPeriod:  conf.AdvancedMetrics.PublishingPeriod,
		},
		TableSizesLimits: conf.AdvancedMetrics.TableSizesLimits,
	}

	checkAdvancedMetricsDefaults(&cfg)

	schema, err := builder.Build()
	if err != nil {
		log.Warnf("Unable to build schema for Advanced Metrics %v", err)
	}
	app, err := advanced_metrics.NewAdvancedMetrics(cfg, schema)
	if err != nil {
		log.Warnf("Unable to initiate advanced metrics module %v", err)
	}

	return &AdvancedMetrics{
		cfg:              cfg,
		advanced_metrics: app,
		commonDims:       metrics.NewCommonDim(env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false), conf, ""),
	}
}

func (m *AdvancedMetrics) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", advancedMetricsPluginName)
	m.pipeline = pipeline
	ctx, cancel := context.WithCancel(m.pipeline.Context())
	m.ctx = ctx
	m.ctxCancel = cancel
	go m.run()
}

func (m *AdvancedMetrics) Close() {
	log.Infof("%s is wrapping up", advancedMetricsPluginName)
	m.ctxCancel()
}

func (*AdvancedMetrics) Process(_ *core.Message) {}

func (m *AdvancedMetrics) run() {
	go func() {
		err := m.advanced_metrics.Run(m.ctx)
		if err != nil {
			log.Errorf("%s failed: %s", advancedMetricsPluginName, err.Error())
		}
	}()
	defer m.ctxCancel()
	err := enableWritePermissionForSocket(m.cfg.Address)
	if err != nil {
		log.Error("App centric metric plugin failed to change socket permissions")
	}
	commonDimensions := append(m.commonDims.ToDimensions(), &proto.Dimension{
		Name:  aggregationDurationDimension,
		Value: strconv.Itoa(int(m.cfg.PublishingPeriod.Seconds())),
	})
	for {
		select {
		case mr, ok := <-m.advanced_metrics.OutChannel():
			if !ok {
				log.Errorf("App centric metric channel unexpectedly closed")
				return
			}
			now := types.TimestampNow()
			m.pipeline.Process(core.NewMessage(core.CommMetrics, []core.Payload{toMetricReport(mr, now, commonDimensions)}))
		case <-m.pipeline.Context().Done():
			return
		}
	}
}

func enableWritePermissionForSocket(path string) error {
	timeout := time.After(time.Second * 1)
	var lastError error
	for {
		select {
		case <-timeout:
			return lastError
		default:
			lastError = os.Chmod(path, 0660)
			if lastError == nil {
				return nil
			}
		}
		<-time.After(time.Microsecond * 100)
	}
}

func toMetricReport(set []*publisher.MetricSet, now *types.Timestamp, commonDimensions []*proto.Dimension) *proto.MetricsReport {
	mr := &proto.MetricsReport{
		Meta: &proto.Metadata{Timestamp: now},
		Type: proto.MetricsReport_INSTANCE,
		Data: make([]*proto.StatsEntity, 0, len(set)),
	}

	for _, s := range set {
		statsEntity := proto.StatsEntity{
			Timestamp:     now,
			Simplemetrics: make([]*proto.SimpleMetric, 0, len(s.Metrics)*4),
			Dimensions:    commonDimensions,
		}

		isStreamMetric := false
		for d := range s.Dimensions {
			statsEntity.Dimensions = append(statsEntity.Dimensions, &proto.Dimension{
				Name:  s.Dimensions[d].Name,
				Value: s.Dimensions[d].Value,
			})
			if s.Dimensions[d].Name == familyDimension &&
				s.Dimensions[d].Value == streamMetricFamilyDimensionValue {
				isStreamMetric = true
			}
		}

		metricNamePrefix := ""
		if isStreamMetric {
			metricNamePrefix = streamMetricPrefix
		} else {
			metricNamePrefix = httpMetricPrefix
		}

		for i := range s.Metrics {
			metricName := s.Metrics[i].Name
			if _, ok := maxOnlyMetrics[metricName]; ok {
				statsEntity.Simplemetrics = append(statsEntity.Simplemetrics, &proto.SimpleMetric{
					Name:  metricName + ".max",
					Value: s.Metrics[i].Values.Max,
				})
			}
			if metricName == hitcountMetric {
				name := metricNamePrefix
				if isStreamMetric {
					name += ".connections"
				} else {
					name += ".count"
				}
				statsEntity.Simplemetrics = append(statsEntity.Simplemetrics, &proto.SimpleMetric{
					Name:  name,
					Value: s.Metrics[i].Values.Count,
				})
			}
			if _, ok := totalOnlyMetrics[metricName]; ok {
				statsEntity.Simplemetrics = append(statsEntity.Simplemetrics, &proto.SimpleMetric{
					Name:  metricNamePrefix + "." + metricName,
					Value: s.Metrics[i].Values.Sum,
				})
			}
		}
		mr.Data = append(mr.Data, &statsEntity)
	}
	return mr
}

func (m *AdvancedMetrics) Info() *core.Info {
	return core.NewInfo(advancedMetricsPluginName, advancedMetricsPluginVersion)
}

func (m *AdvancedMetrics) Subscriptions() []string {
	return []string{}
}

func checkAdvancedMetricsDefaults(cfg *advanced_metrics.Config) {
	config.CheckAndSetDefault(&cfg.Address, config.Defaults.AdvancedMetrics.SocketPath)
	config.CheckAndSetDefault(&cfg.AggregationPeriod, config.Defaults.AdvancedMetrics.AggregationPeriod)
	config.CheckAndSetDefault(&cfg.PublishingPeriod, config.Defaults.AdvancedMetrics.PublishingPeriod)
	config.CheckAndSetDefault(&cfg.TableSizesLimits.StagingTableMaxSize, config.Defaults.AdvancedMetrics.TableSizesLimits.StagingTableMaxSize)
	config.CheckAndSetDefault(&cfg.TableSizesLimits.StagingTableThreshold, config.Defaults.AdvancedMetrics.TableSizesLimits.StagingTableThreshold)
	config.CheckAndSetDefault(&cfg.TableSizesLimits.PriorityTableMaxSize, config.Defaults.AdvancedMetrics.TableSizesLimits.PriorityTableMaxSize)
	config.CheckAndSetDefault(&cfg.TableSizesLimits.PriorityTableThreshold, config.Defaults.AdvancedMetrics.TableSizesLimits.PriorityTableThreshold)
}
