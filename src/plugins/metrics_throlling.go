/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */
package plugins

import (
	"context"
	"sync"
	"time"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

const (
	reportStaggeringStartTime = 5555 * time.Millisecond // want to start the report cycle staggering of the collection time
)

type MetricsThrottle struct {
	messagePipeline    core.MessagePipeInterface
	BulkSize           int
	metricBuffer       []core.Payload
	ticker             *time.Ticker
	reportsReady       *atomic.Bool
	collectorsUpdate   *atomic.Bool
	metricsAggregation bool
	metricsCollections map[proto.MetricsReport_Type]*metrics.Collections
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	mu                 sync.Mutex
	env                core.Environment
	conf               *config.Config
	errors             chan error
}

func NewMetricsThrottle(conf *config.Config, env core.Environment) *MetricsThrottle {
	return &MetricsThrottle{
		metricBuffer:       make([]core.Payload, 0),
		BulkSize:           conf.AgentMetrics.BulkSize,
		ticker:             time.NewTicker(conf.AgentMetrics.ReportInterval + reportStaggeringStartTime),
		reportsReady:       atomic.NewBool(false),
		collectorsUpdate:   atomic.NewBool(false),
		metricsAggregation: conf.AgentMetrics.Mode == "aggregated",
		metricsCollections: make(map[proto.MetricsReport_Type]*metrics.Collections, 0),
		env:                env,
		conf:               conf,
		errors:             make(chan error),
	}
}

func (r *MetricsThrottle) Init(pipeline core.MessagePipeInterface) {
	r.messagePipeline = pipeline
	r.ctx, r.cancel = context.WithCancel(pipeline.Context())
	if r.metricsAggregation {
		r.wg.Add(1)
		go r.metricsReportGoroutine()
	}
	log.Info("MetricsThrottle initializing")
}

func (r *MetricsThrottle) Close() {
	log.Info("MetricsThrottle is wrapping up")
	r.reportsReady.Store(false) // allow metricsReportGoroutine to shutdown gracefully
	r.cancel()
	r.wg.Wait()
	r.ticker.Stop()
}

func (r *MetricsThrottle) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureMetricsThrottle, "v0.0.1")
}

func (r *MetricsThrottle) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.AgentConfigChanged):
		// If the agent config on disk changed update MetricsThrottle with relevant config info
		r.syncAgentConfigChange()
		r.collectorsUpdate.Store(true)
		return

	case msg.Exact(core.MetricReport):
		if r.metricsAggregation {
			switch bundle := msg.Data().(type) {
			case *metrics.MetricsReportBundle:
				if len(bundle.Data) > 0 {
					r.mu.Lock()
					for _, report := range bundle.Data {
						if len(report.Data) > 0 {
							if _, ok := r.metricsCollections[report.Type]; !ok {
								r.metricsCollections[report.Type] = &metrics.Collections{
									Count:        0,
									MetricsCount: make(map[string]metrics.PerDimension),
									Data:         make(map[string]metrics.PerDimension),
								}
							}
							collection := metrics.SaveCollections(*r.metricsCollections[report.Type], report)
							r.metricsCollections[report.Type] = &collection
							log.Debugf("MetricsThrottle: Metrics collection saved [Type: %d]", report.Type)
						}
					}
					r.mu.Unlock()
					r.reportsReady.Store(true)
				}
			}
		} else {
			switch bundle := msg.Data().(type) {
			case *metrics.MetricsReportBundle:
				if len(bundle.Data) > 0 {
					for _, report := range bundle.Data {
						if len(report.Data) > 0 {
							r.metricBuffer = append(r.metricBuffer, report)
						}
					}
				}
			}
			log.Tracef("MetricsThrottle buffer size: %d of %d", len(r.metricBuffer), r.BulkSize)
			if len(r.metricBuffer) >= r.BulkSize {
				log.Info("MetricsThrottle buffer flush")
				r.messagePipeline.Process(core.NewMessage(core.CommMetrics, r.metricBuffer))
				r.metricBuffer = make([]core.Payload, 0)
			}
		}
	}
}

func (r *MetricsThrottle) Subscriptions() []string {
	return []string{core.MetricReport, core.AgentConfigChanged}
}

func (r *MetricsThrottle) metricsReportGoroutine() {
	defer r.wg.Done()
	defer r.ticker.Stop()
	defer close(r.errors)
	log.Info("MetricsThrottle waiting for report ready")
	for {
		select {
		case <-r.ctx.Done():
			err := r.ctx.Err()
			if err != nil && err != context.Canceled {
				log.Errorf("error in done context metricsReportGoroutine %v", err)
			}
			return
		default:
			if !r.reportsReady.Load() {
				continue
			}
		}

		select {
		case <-r.ctx.Done():
			err := r.ctx.Err()
			if err != nil && err != context.Canceled {
				log.Errorf("error in done context metricsReportGoroutine %v", err)
			}
			return
		case <-r.ticker.C:
			reports := r.getAggregatedReports()
			log.Debugf("metricsThrottle: metricsReportGoroutine, got %d reports to send", len(reports))
			if len(reports) > 0 {
				r.messagePipeline.Process(core.NewMessage(core.CommMetrics, reports))
			}
			if r.collectorsUpdate.Load() {
				r.BulkSize = r.conf.AgentMetrics.BulkSize
				r.metricsAggregation = r.conf.AgentMetrics.Mode == "aggregated"
				r.ticker.Stop()
				r.ticker = time.NewTicker(r.conf.AgentMetrics.ReportInterval + reportStaggeringStartTime)
				r.messagePipeline.Process(core.NewMessage(core.AgentCollectorsUpdate, ""))
				r.collectorsUpdate.Store(false)
			}
		case err := <-r.errors:
			if err != nil {
				log.Errorf("Error in metricsReportGoroutine %v", err)
			}
		}
	}
}

func (r *MetricsThrottle) syncAgentConfigChange() {
	conf, err := config.GetConfig(r.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	if conf.DisplayName == "" {
		conf.DisplayName = r.env.GetHostname()
		log.Infof("setting displayName to %s", conf.DisplayName)
	}
	log.Debugf("MetricsThrottle is updating to a new config - %v", conf)

	r.conf = conf
}

func (r *MetricsThrottle) getAggregatedReports() (reports []core.Payload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for reportType, collection := range r.metricsCollections {
		report := &proto.MetricsReport{
			Meta: &proto.Metadata{
				Timestamp: types.TimestampNow(),
			},
			Type: reportType,
			Data: metrics.GenerateMetrics(*collection),
		}

		reports = append(reports, report)

		r.metricsCollections[reportType] = &metrics.Collections{
			Count:        0,
			MetricsCount: map[string]metrics.PerDimension{},
			Data:         make(map[string]metrics.PerDimension),
		}
	}

	return reports
}
