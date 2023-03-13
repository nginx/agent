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
}

func (pf *PhpFpm) Process(msg *core.Message) {
   return core.NewMessage(core.PhpFpmMetrics, readPhpFmpStatus())

}

func (pf *PhpFpm) Subscriptions() []string {
	return []string{}
}

func (pf *PhpFpm) Close() {
	log.Infof("%s is wrapping up", napMonitoringPluginName)
}
