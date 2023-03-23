/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"context"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/payloads"
	php_fpm "github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/php-fpm-metrics"
	log "github.com/sirupsen/logrus"
)

const (
	phpFpmMetricsPluginVersion = "v0.1.0"
	phpFpmMetricsPluginName    = "PHP-FPM Metrics"
	phpConnAcceptedMetric      = "php.fpm.conn.accepted"
	phpQueueCurrentMetric      = "php.fpm.queue.current"
	phpQueueMaxMetric          = "php.fpm.queue.max"
	phpQueueLenMetric          = "php.fpm.queue.len"
	phpProcIdleMetric          = "php.fpm.proc.idle"
	phpProcActiveMetric        = "php.fpm.proc.active"
	phpProcTotalMetric         = "php.fpm.proc.total"
	phpProcMaxActiveMetric     = "php.fpm.proc.max_active"
	phpProcMaxChildMetric      = "php.fpm.proc.max_child"
	phpSlowReqMetric           = "php.fpm.slow_req"
)

type PhpFpmMetricsConfig struct {
	SocketPath string `mapstructure:"socket_path"`
}

type PhpFpmMetrics struct {
	ctx             context.Context
	ctxCancel       context.CancelFunc
	cfg             php_fpm.Config
	php_fpm_metrics *php_fpm.PhpFpmMetrics
	registration    *php_fpm.PhpFpmRegistration
	pipeline        core.MessagePipeInterface
	commonDims      *metrics.CommonDim
	env             core.Environment
	agent           string
}

func NewPhpFpmMetrics(env core.Environment, conf *config.Config, version string, phpFpmMetricsConf interface{}) *PhpFpmMetrics {
	phpFpmMetricsConfig := &PhpFpmMetricsConfig{}

	if phpFpmMetricsConf != nil {
		var err error
		phpFpmMetricsConfig, err = agent_config.DecodeConfig[*PhpFpmMetricsConfig](phpFpmMetricsConf)
		if err != nil {
			log.Errorf("Error decoding configuration for extension plugin %s, %v", phpFpmMetricsPluginName, err)
			return nil
		}
	}

	cfg := php_fpm.Config{
		Address: phpFpmMetricsConfig.SocketPath,
	}

	registration := &php_fpm.PhpFpmRegistration{
		Env: env,
	}
	return &PhpFpmMetrics{
		cfg:             cfg,
		registration:    registration,
		env:             env,
		php_fpm_metrics: &php_fpm.PhpFpmMetrics{},
		agent:           version,
		commonDims:      metrics.NewCommonDim(env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false), conf, ""),
	}
}

func (m *PhpFpmMetrics) Init(pipeline core.MessagePipeInterface) {
	intialphpFpmDetails := m.registration.GeneratePhpFpmDetailsProtoCommand(m.agent)
	m.pipeline = pipeline
	ctx, cancel := context.WithCancel(m.pipeline.Context())
	m.ctx = ctx
	m.ctxCancel = cancel

	m.pipeline.Process(
		core.NewMessage(
			core.DataplaneSoftwareDetailsUpdated,
			payloads.NewDataplaneSoftwareDetailsUpdate(
				phpFpmMetricsPluginName,
				&proto.DataplaneSoftwareDetails{
					Data: intialphpFpmDetails,
				},
			),
		),
	)
}

func (pf *PhpFpmMetrics) Info() *core.Info {
	return core.NewInfo(phpFpmMetricsPluginName, phpFpmMetricsPluginVersion)
}

func (pf *PhpFpmMetrics) Process(msg *core.Message) {
	log.Info("*** PhpFpmMetrics Process() ***")
}

func (pf *PhpFpmMetrics) Subscriptions() []string {
	return []string{}
}

func (m *PhpFpmMetrics) Close() {
	m.ctxCancel()
}
