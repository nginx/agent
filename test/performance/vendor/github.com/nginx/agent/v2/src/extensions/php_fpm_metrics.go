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
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	log "github.com/sirupsen/logrus"
)

const (
	phpFpmMetricsExtensionPluginVersion = "v0.1.0"
)

type PhpFpmMetrics struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	pipeline  core.MessagePipeInterface
	env       core.Environment
}

func NewPhpFpmMetrics(env core.Environment, conf *config.Config) (*PhpFpmMetrics, error) {
	return &PhpFpmMetrics{
		env: env,
	}, nil
}

func (pfm *PhpFpmMetrics) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", agent_config.PhpFpmMetricsExtensionPlugin)
	pfm.pipeline = pipeline
	ctx, cancel := context.WithCancel(pfm.pipeline.Context())
	pfm.ctx = ctx
	pfm.ctxCancel = cancel
}

func (pf *PhpFpmMetrics) Info() *core.Info {
	return core.NewInfo(agent_config.PhpFpmMetricsExtensionPlugin, phpFpmMetricsExtensionPluginVersion)
}

func (pfm *PhpFpmMetrics) Process(msg *core.Message) {
	log.Tracef("Process phpfpm metrics : %v", msg)
}

func (pfm *PhpFpmMetrics) Subscriptions() []string {
	return []string{}
}

func (pfm *PhpFpmMetrics) Close() {
	pfm.ctxCancel()
}
