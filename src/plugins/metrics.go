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

	"github.com/nginx/agent/sdk/v2"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/collectors"

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

type Metrics struct {
	pipeline                 core.MessagePipeInterface
	collectorsUpdate         *atomic.Bool
	ticker                   *time.Ticker
	interval                 time.Duration
	collectors               []metrics.Collector
	buf                      chan *metrics.StatsEntityWrapper
	errors                   chan error
	collectorConfigsMap      map[string]*metrics.NginxCollectorConfig
	ctx                      context.Context
	wg                       sync.WaitGroup
	collectorsMutex          sync.RWMutex
	collectorConfigsMapMutex sync.Mutex
	env                      core.Environment
	conf                     *config.Config
	binary                   core.NginxBinary
	processesMutex           sync.RWMutex
	processes                []*core.Process
}

func NewMetrics(config *config.Config, env core.Environment, binary core.NginxBinary, processes []*core.Process) *Metrics {
	collectorConfigsMap := createCollectorConfigsMap(config, env, binary, processes)
	return &Metrics{
		collectorsUpdate:         atomic.NewBool(false),
		ticker:                   time.NewTicker(config.AgentMetrics.CollectionInterval),
		interval:                 config.AgentMetrics.CollectionInterval,
		buf:                      make(chan *metrics.StatsEntityWrapper, 4096),
		errors:                   make(chan error),
		collectorConfigsMap:      collectorConfigsMap,
		wg:                       sync.WaitGroup{},
		collectorsMutex:          sync.RWMutex{},
		collectorConfigsMapMutex: sync.Mutex{},
		env:                      env,
		conf:                     config,
		binary:                   binary,
		processes:                processes,
	}
}

func (m *Metrics) Init(pipeline core.MessagePipeInterface) {
	log.Info("Metrics initializing")
	m.pipeline = pipeline
	m.ctx = pipeline.Context()
	go m.metricsGoroutine()
}

func (m *Metrics) Close() {
	m.collectors = nil
	log.Info("Metrics is wrapping up")
}

func (m *Metrics) Process(msg *core.Message) {
	log.Debugf("Process function in the metrics.go, %s %v", msg.Topic(), msg.Data())
	switch {
	case msg.Exact(core.AgentConfigChanged), msg.Exact(core.NginxConfigApplySucceeded):
		// If the agent config on disk changed or the NGINX statusAPI was updated
		// Then update Metrics with relevant config info
		collectorConfigsMap := createCollectorConfigsMap(m.conf, m.env, m.binary, m.getNginxProccessInfo())
		m.collectorConfigsMapMutex.Lock()
		m.collectorConfigsMap = collectorConfigsMap
		m.collectorConfigsMapMutex.Unlock()

		m.syncAgentConfigChange()
		m.updateCollectorsConfig()
		return

	case msg.Exact(core.AgentCollectorsUpdate):
		// update collectors and collection time intervals after the report cycle triggered
		m.collectorsUpdate.Store(true)
		m.updateCollectorsConfig()
		return

	case msg.Exact(core.NginxConfigApplySucceeded):
		m.updateCollectorsSources()
		return

	case msg.Exact(core.NginxDetailProcUpdate):
		m.syncProcessInfo(msg.Data().([]*core.Process))
		collectorConfigsMap := createCollectorConfigsMap(m.conf, m.env, m.binary, m.getNginxProccessInfo())
		for key, collectorConfig := range collectorConfigsMap {
			if _, ok := m.collectorConfigsMap[key]; !ok {
				log.Debugf("Adding new nginx collector for nginx id: %s", collectorConfig.NginxId)
				m.collectors = append(m.collectors,
					collectors.NewNginxCollector(m.conf, m.env, collectorConfig, m.binary),
				)
			}
		}

		collectorsToStop := []string{}
		for key, collectorConfig := range m.collectorConfigsMap {
			if _, ok := collectorConfigsMap[key]; !ok {
				collectorsToStop = append(collectorsToStop, collectorConfig.NginxId)
			}
		}

		m.collectorConfigsMapMutex.Lock()
		m.collectorConfigsMap = collectorConfigsMap
		m.collectorConfigsMapMutex.Unlock()

		stoppedCollectorIndex := -1

		m.collectorsMutex.RLock()
		for index, collector := range m.collectors {
			if nginxCollector, ok := collector.(*collectors.NginxCollector); ok {
				for _, nginxId := range collectorsToStop {
					if nginxCollector.GetNginxId() == nginxId {
						stoppedCollectorIndex = index
						nginxCollector.Stop()
						log.Debugf("Removing nginx collector for nginx id: %s", nginxCollector.GetNginxId())
						break
					}
				}
			}
		}
		m.collectorsMutex.RUnlock()

		if stoppedCollectorIndex >= 0 {
			m.collectors = append(m.collectors[:stoppedCollectorIndex], m.collectors[stoppedCollectorIndex+1:]...)
		}
		return
	}
}

func (m *Metrics) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureMetrics, "v0.0.2")
}

func (m *Metrics) Subscriptions() []string {
	return []string{
		core.RegistrationCompletedTopic,
		core.AgentCollectorsUpdate,
		core.AgentConfigChanged,
		core.NginxPluginConfigured,
		core.NginxDetailProcUpdate,
		core.NginxConfigApplySucceeded,
	}
}

func (m *Metrics) metricsGoroutine() {
	m.wg.Add(1)
	defer m.ticker.Stop()
	defer m.wg.Done()
	log.Info("Metrics waiting for handshake to be completed")
	m.registerStatsSources()

	for {
		select {
		case <-m.ctx.Done():
			err := m.ctx.Err()
			if err != nil {
				log.Errorf("error in done context metricsGoroutine %v", err)
			}
			return
		case <-m.ticker.C:
			stats := m.collectStats()
			if bundlePayload := metrics.GenerateMetricsReportBundle(stats); bundlePayload != nil {
				if m.conf.IsFeatureEnabled(agent_config.FeatureMetrics) || m.conf.IsFeatureEnabled(agent_config.FeatureMetricsThrottle) {
					m.pipeline.Process(core.NewMessage(core.MetricReport, bundlePayload))
				} else {
					metricBuffer := make([]core.Payload, 0)

					switch bundle := bundlePayload.(type) {
					case *metrics.MetricsReportBundle:
						if len(bundle.Data) > 0 {
							for _, report := range bundle.Data {
								if len(report.Data) > 0 {
									metricBuffer = append(metricBuffer, report)
								}
							}
						}
						m.pipeline.Process(core.NewMessage(core.CommMetrics, metricBuffer))
					default:
						log.Errorf("Error converting metric report: %T", bundlePayload)
					}
				}
			}

			if m.collectorsUpdate.Load() {
				m.ticker = time.NewTicker(m.conf.AgentMetrics.CollectionInterval)
				m.collectorsUpdate.Store(false)
			}

		case err := <-m.errors:
			log.Errorf("Error in metricsGoroutine %v", err)
		}
	}
}

func (m *Metrics) collectStats() (stats []*metrics.StatsEntityWrapper) {
	// setups a collect duration of half-time of the poll interval
	ctx, cancel := context.WithTimeout(m.ctx, m.interval/2)
	defer cancel()
	// locks the m.collectors to make sure it doesn't get deleted in the middle
	// of collection, as we will delete the old one if config changes.
	// maybe we can fine tune the lock later, but the collection has been very quick so far.
	m.collectorsMutex.Lock()
	defer m.collectorsMutex.Unlock()
	wg := &sync.WaitGroup{}
	start := time.Now()
	for _, s := range m.collectors {
		wg.Add(1)
		go s.Collect(ctx, wg, m.buf)
	}
	// wait until all the collection go routines are done, which either context timeout or exit
	wg.Wait()

	for len(m.buf) > 0 {
		// drain the buf, since our sources/collectors are all done, we can rely on buffer length
		select {
		case <-ctx.Done():
			err := m.ctx.Err()
			if err != nil {
				log.Errorf("error in done context collectStats %v", err)
			}
			return
		case stat := <-m.buf:
			stats = append(stats, stat)
		}
	}

	log.Debugf("collected %d entries in %s (ctx error=%t)", len(stats), time.Since(start), ctx.Err() != nil)
	return
}

func (m *Metrics) registerStatsSources() {
	tempCollectors := make([]metrics.Collector, 0)

	if m.conf.IsFeatureEnabled(agent_config.FeatureMetrics) || m.conf.IsFeatureEnabled(agent_config.FeatureMetricsCollection) {
		tempCollectors = append(tempCollectors,
			collectors.NewSystemCollector(m.env, m.conf),
		)

		if m.env.IsContainer() {
			tempCollectors = append(tempCollectors,
				collectors.NewContainerCollector(m.env, m.conf),
			)
		}
	}

	hasNginxCollector := false
	m.collectorConfigsMapMutex.Lock()
	for key := range m.collectorConfigsMap {
		tempCollectors = append(tempCollectors,
			collectors.NewNginxCollector(m.conf, m.env, m.collectorConfigsMap[key], m.binary),
		)
		hasNginxCollector = true
	}
	m.collectorConfigsMapMutex.Unlock()

	// if NGINX is not running/detected, still run the static collector to output nginx.status = 0.
	if !hasNginxCollector {
		// Just use the default NGINX process path and default NGINX config path to create the NginxID.
		nginxID := core.GenerateNginxID("%s_%s_%s", "/usr/sbin/nginx", "/etc/nginx/nginx.conf", "prefix")
		tempCollectors = append(tempCollectors,
			collectors.NewNginxCollector(m.conf, m.env, &metrics.NginxCollectorConfig{NginxId: nginxID}, m.binary),
		)
	}

	m.collectorsMutex.Lock()
	m.collectors = tempCollectors
	m.collectorsMutex.Unlock()
}

func (m *Metrics) syncAgentConfigChange() {
	conf, err := config.GetConfig(m.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	log.Debugf("Metrics is updating to a new config - %+v", conf)

	if conf.DisplayName == "" {
		conf.DisplayName = m.env.GetHostname()
		log.Infof("setting displayName to %s", conf.DisplayName)
	}

	// Update Metrics with relevant config info
	m.conf = conf
}

func createCollectorConfigsMap(config *config.Config, env core.Environment, binary core.NginxBinary, processes []*core.Process) map[string]*metrics.NginxCollectorConfig {
	collectorConfigsMap := make(map[string]*metrics.NginxCollectorConfig)

	for _, p := range processes {
		if !p.IsMaster {
			continue
		}
		detail := binary.GetNginxDetailsFromProcess(p)

		stubStatusApi, plusApi := "", ""
		if detail.Plus.Enabled {
			plusApi = detail.StatusUrl
		} else {
			stubStatusApi = detail.StatusUrl
		}

		errorLogs, accessLogs, err := sdk.GetErrorAndAccessLogsWithIgnoreDirectives(detail.ConfPath, config.IgnoreDirectives)
		if err != nil {
			log.Warnf("Error reading access and error logs from config %s %v", detail.ConfPath, err)
		}

		collectorConfigsMap[detail.NginxId] = &metrics.NginxCollectorConfig{
			StubStatus:         stubStatusApi,
			PlusAPI:            plusApi,
			BinPath:            detail.ProcessPath,
			ConfPath:           detail.ConfPath,
			CollectionInterval: config.AgentMetrics.CollectionInterval,
			AccessLogs:         sdk.GetAccessLogs(accessLogs),
			ErrorLogs:          sdk.GetErrorLogs(errorLogs),
			NginxId:            detail.NginxId,
			ClientVersion:      config.Nginx.NginxClientVersion,
		}
	}
	return collectorConfigsMap
}

func (m *Metrics) updateCollectorsConfig() {
	log.Trace("Updating collector config")
	for _, collector := range m.collectors {
		if nginxCollector, ok := collector.(*collectors.NginxCollector); ok {
			if collectorConfig, ok := m.collectorConfigsMap[nginxCollector.GetNginxId()]; ok {
				log.Tracef("Updating nginx collector config for nginxId %s", collectorConfig.NginxId)
				nginxCollector.UpdateCollectorConfig(collectorConfig, m.conf, m.env)
			}
		}
		collector.UpdateConfig(m.conf)
	}
}

func (m *Metrics) updateCollectorsSources() {
	log.Trace("Updating nginx collector sources")
	for _, collector := range m.collectors {
		if nginxCollector, ok := collector.(*collectors.NginxCollector); ok {
			nginxCollector.UpdateSources()
		}
	}
}

func (m *Metrics) getNginxProccessInfo() []*core.Process {
	m.processesMutex.RLock()
	defer m.processesMutex.RUnlock()
	return m.processes
}

func (m *Metrics) syncProcessInfo(processInfo []*core.Process) {
	m.processesMutex.Lock()
	defer m.processesMutex.Unlock()
	m.processes = processInfo
}
