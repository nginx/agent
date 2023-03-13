/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	phpFpmMetricsPluginName    = agent_config.PhpFpmMetericsExtensionPlugin
)

type PhpFpm struct {

}


func (pf *PhpFpm) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", phpFpmMetricsPluginName)
}

func (pf *PhpFpm) Info() {
	log.Info("phpfpm info")
	log.Info("phpfpm info")
	log.Info("phpfpm info")
}

func (pf *PhpFpm) Process(msg *core.Message) {
	log.Info("phpfpm process")
	log.Info("phpfpm process")
	log.Info("phpfpm process")
}

func (pf *PhpFpm) Subscriptions() []string {
	return []string{}
}

func (pf *PhpFpm) Close() {
	log.Infof("%s is wrapping up", phpFpmMetricsPluginName)
}

type PhpFpmMetrics struct {
}

func NewPhpFpmMetrics(env core.Environment, conf *config.Config, advancedMetricsConf interface{}) *PhpFpmMetrics {
	// php fpm installed?
	
	return &PhpFpmMetrics{}
}

