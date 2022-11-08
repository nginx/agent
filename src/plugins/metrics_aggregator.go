package plugins

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
)

const (
	reportStaggeringStartTime = 5555 * time.Millisecond // want to start the report cycle staggering of the collection time
)

type MetricsAggregator struct {
	messagePipeline    core.MessagePipeInterface
	BulkSize           int
	metricBuffer       []core.Payload
	ticker             *time.Ticker
	reportsReady       *atomic.Bool
	collectorsUpdate   *atomic.Bool
	metricsCollections metrics.Collections
	ctx                context.Context
	wg                 sync.WaitGroup
	mu                 sync.Mutex
	env                core.Environment
	conf               *config.Config
	errors             chan error
}

func NewMetricsAggregator(conf *config.Config, env core.Environment) *MetricsAggregator {
	metricsCollections := metrics.Collections{
		Count: 0,
		Data:  make(map[string]metrics.PerDimension),
	}

	return &MetricsAggregator{
		metricBuffer:       make([]core.Payload, 0),
		BulkSize:           conf.AgentMetrics.BulkSize,
		ticker:             time.NewTicker(conf.AgentMetrics.ReportInterval + reportStaggeringStartTime),
		reportsReady:       atomic.NewBool(false),
		collectorsUpdate:   atomic.NewBool(false),
		metricsCollections: metricsCollections,
		wg:                 sync.WaitGroup{},
		env:                env,
		conf:               conf,
		errors:             make(chan error),
	}
}

func (r *MetricsAggregator) Init(pipeline core.MessagePipeInterface) {
	r.messagePipeline = pipeline
	r.ctx = pipeline.Context()
	r.wg.Add(1)
	go r.metricsReportGoroutine(r.ctx, &r.wg)
	log.Info("MetricsAggregator initializing")
}

func (r *MetricsAggregator) Close() {
	log.Info("MetricsAggregator is wrapping up")
	r.reportsReady.Store(false) // allow metricsReportGoroutine to shutdown gracefully
	r.ticker.Stop()
}

func (r *MetricsAggregator) Info() *core.Info {
	return core.NewInfo("MetricsAggregator", "v0.0.1")
}

func (r *MetricsAggregator) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.AgentConfigChanged):
		// If the agent config on disk changed update MetricsAggregator with relevant config info
		r.syncAgentConfigChange()
		r.collectorsUpdate.Store(true)
		return
	case msg.Exact(core.MetricReport):
		switch report := msg.Data().(type) {
		case *proto.MetricsReport:
			r.mu.Lock()
			r.metricsCollections = metrics.SaveCollections(r.metricsCollections, report)
			r.mu.Unlock()
			log.Debug("MetricsAggregator: Metrics collection saved")
			r.reportsReady.Store(true)
		}
	}
}

func (r *MetricsAggregator) Subscriptions() []string {
	return []string{core.MetricReport, core.AgentConfigChanged, core.LoggerLevel}
}

func (r *MetricsAggregator) metricsReportGoroutine(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer r.ticker.Stop()
	log.Info("MetricsAggregator waiting for report ready")
	for {
		if !r.reportsReady.Load() {
			continue
		}

		select {
		case <-ctx.Done():
			err := r.ctx.Err()
			if err != nil {
				log.Errorf("error in done context metricsReportGoroutine %v", err)
			}
			return
		case <-r.ticker.C:
			aggregatedReport := r.getAggregatedReport()
			r.messagePipeline.Process(
				core.NewMessage(core.CommMetrics, []core.Payload{aggregatedReport}),
			)
			if r.collectorsUpdate.Load() {
				r.BulkSize = r.conf.AgentMetrics.BulkSize
				r.ticker.Stop()
				r.ticker = time.NewTicker(r.conf.AgentMetrics.ReportInterval + reportStaggeringStartTime)
				r.messagePipeline.Process(core.NewMessage(core.AgentCollectorsUpdate, ""))
				r.collectorsUpdate.Store(false)
			}
		case err := <-r.errors:
			log.Errorf("Error in metricsReportGoroutine %v", err)
		}
	}
}

func (r *MetricsAggregator) syncAgentConfigChange() {
	conf, err := config.GetConfig(r.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	if conf.DisplayName == "" {
		conf.DisplayName = r.env.GetHostname()
		log.Infof("setting displayName to %s", conf.DisplayName)
	}
	log.Debugf("MetricsAggregator is updating to a new config - %v", conf)

	r.conf = conf
}

func (r *MetricsAggregator) getAggregatedReport() *proto.MetricsReport {
	r.mu.Lock()
	defer r.mu.Unlock()
	report := metrics.GenerateMetricsReport(r.metricsCollections)
	r.metricsCollections = metrics.Collections{
		Count: 0,
		Data:  make(map[string]metrics.PerDimension),
	}
	return report
}
