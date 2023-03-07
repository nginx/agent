/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/payloads"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
)

const (
	napMonitoringPluginName    = agent_config.NginxAppProtectMonitoringExtensionPlugin
	napMonitoringPluginVersion = "v0.0.1"
	minReportIntervalDelimiter = time.Minute
	minReportCountDelimiter    = 1
	maxReportCountDelimiter    = 400
)

var nginxAppProtectMonitoringDefault = &manager.NginxAppProtectMonitoringConfig{
	ProcessorBufferSize: 50000,
	CollectorBufferSize: 50000,
	SyslogIP:            "0.0.0.0",
	SyslogPort:          514,
	ReportInterval:      time.Minute,
	ReportCount:         400,
}

type NAPMonitoring struct {
	monitorMgr      *manager.Manager
	messagePipeline core.MessagePipeInterface
	reportInterval  time.Duration
	reportCount     int
	ctx             context.Context
	ctxCancel       context.CancelFunc
}

func NewNAPMonitoring(env core.Environment, cfg *config.Config, nginxAppProtectMonitoringConf interface{}) (*NAPMonitoring, error) {
	commonDims := metrics.NewCommonDim(env.NewHostInfo("agentVersion", &cfg.Tags, cfg.ConfigDirs, false), cfg, "")
	nginxAppProtectMonitoringConfig := nginxAppProtectMonitoringDefault

	if nginxAppProtectMonitoringConf != nil {
		var err error
		nginxAppProtectMonitoringConfig, err = agent_config.DecodeConfig[*manager.NginxAppProtectMonitoringConfig](nginxAppProtectMonitoringConf)
		if err != nil {
			log.Errorf("Error decoding configuration for extension plugin %s, %v", napMonitoringPluginName, err)
			return nil, err
		}
	}

	m, err := manager.NewManager(nginxAppProtectMonitoringConfig, commonDims)
	if err != nil {
		return nil, err
	}

	if !(nginxAppProtectMonitoringConfig.ReportInterval > minReportIntervalDelimiter) {
		log.Warnf("NAP Monitoring report interval must be higher than %v. Defaulting to %v",
			minReportIntervalDelimiter,
			nginxAppProtectMonitoringDefault.ReportInterval)
		nginxAppProtectMonitoringConfig.ReportInterval = nginxAppProtectMonitoringDefault.ReportInterval
	}
	if nginxAppProtectMonitoringConfig.ReportCount < minReportCountDelimiter ||
		nginxAppProtectMonitoringConfig.ReportCount > maxReportCountDelimiter {
		log.Warnf("NAP Monitoring report count must be between %v and %v. Defaulting to %v",
			minReportCountDelimiter,
			maxReportCountDelimiter,
			nginxAppProtectMonitoringDefault.ReportCount)
		nginxAppProtectMonitoringConfig.ReportCount = nginxAppProtectMonitoringDefault.ReportCount
	}

	return &NAPMonitoring{
		monitorMgr:     m,
		reportInterval: nginxAppProtectMonitoringConfig.ReportInterval,
		reportCount:    nginxAppProtectMonitoringConfig.ReportCount,
	}, nil
}

func (n *NAPMonitoring) Info() *core.Info {
	return core.NewInfo(napMonitoringPluginName, napMonitoringPluginVersion)
}

func (n *NAPMonitoring) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", napMonitoringPluginName)
	n.messagePipeline = pipeline
	ctx, cancel := context.WithCancel(n.messagePipeline.Context())
	n.ctx = ctx
	n.ctxCancel = cancel

	n.messagePipeline.Process(
		core.NewMessage(
			core.DataplaneSoftwareDetailsUpdated,
			payloads.NewDataplaneSoftwareDetailsUpdate(
				napMonitoringPluginName,
				&proto.DataplaneSoftwareDetails{},
			),
		),
	)

	go n.monitorMgr.Run(ctx)
	go n.run()
}

func (n *NAPMonitoring) Process(msg *core.Message) {}

func (n *NAPMonitoring) Subscriptions() []string {
	return []string{}
}

func (n *NAPMonitoring) Close() {
	log.Infof("%s is wrapping up", napMonitoringPluginName)
	n.ctxCancel()
}

func (n *NAPMonitoring) run() {
	defer n.Close()

	riTicker := time.NewTicker(n.reportInterval)
	defer riTicker.Stop()

	report := &models.EventReport{
		Events: []*models.Event{},
	}

	for {
		select {
		case event, ok := <-n.monitorMgr.OutChannel():
			if !ok {
				log.Errorf("NAP Monitoring processing channel closed unexpectedly")
				return
			}
			report.Events = append(report.Events, event)
			if len(report.Events) == n.reportCount {
				log.Infof("collected %d Security Violation Events, sending report", n.reportCount)
				n.send(report)
			}
		case <-riTicker.C:
			if len(report.Events) > 0 {
				log.Infof("reached a report interval of %vs, sending %d Security Violation Events as a report", n.reportInterval.Seconds(), len(report.Events))
				n.send(report)
			}
		case <-n.ctx.Done():
			return
		}
	}
}

func (n *NAPMonitoring) send(report *models.EventReport) {
	reportToSend := &models.EventReport{
		Events: make([]*models.Event, len(report.Events)),
	}
	copy(reportToSend.Events, report.Events)
	n.messagePipeline.Process(core.NewMessage(core.CommMetrics, []core.Payload{reportToSend}))
	report.Events = []*models.Event{}
}
