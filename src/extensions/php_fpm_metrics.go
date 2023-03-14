/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	log "github.com/sirupsen/logrus"

	//agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	phpFpmMetricsPluginVersion = "v0.0.0"
	PhpFpmMetricsPluginName    = "php-fpm-metrics"
)

type PhpFpmMetrics struct {
}

type PhpFpmMetricsConfig struct {
}

func (pf *PhpFpmMetrics) Init(pipeline core.MessagePipeInterface) {
	log.Infof("****** %s initializing", PhpFpmMetricsPluginName)
}

func (pf *PhpFpmMetrics) Info() *core.Info {
	log.Info("*** PPhpFpmMetrics Info ***")
	return core.NewInfo(PhpFpmMetricsPluginName, phpFpmMetricsPluginVersion)
}

func (pf *PhpFpmMetrics) Process(msg *core.Message) {
	log.Info("*** PhpFpmMetrics Process ***")
}

func (pf *PhpFpmMetrics) Subscriptions() []string {
	log.Infof("*** phpfpm metrics Subscriptions ***")
	return []string{}
}

func (m *PhpFpmMetrics) Close() {
	log.Infof("*** %s is wrapping up *** P", AdvancedMetricsPluginName)
}

func NewPhpFpmMetrics(env core.Environment, conf *config.Config, phpFpmMetricsConf interface{}) *PhpFpmMetrics {
	// php fpm installed?
	log.Info("*** NewPhpFpmMetrics **")
	return &PhpFpmMetrics{}
}
