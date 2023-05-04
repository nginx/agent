/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
)

type MetricsSender struct {
	reporter    client.MetricReporter
	pipeline    core.MessagePipeInterface
	ctx         context.Context
	started     *atomic.Bool
	readyToSend *atomic.Bool
}

func NewMetricsSender(reporter client.MetricReporter) *MetricsSender {
	return &MetricsSender{
		reporter:    reporter,
		started:     atomic.NewBool(false),
		readyToSend: atomic.NewBool(false),
	}
}

func (r *MetricsSender) Init(pipeline core.MessagePipeInterface) {
	if r.started.Load() {
		return
	}
	r.started.Toggle()
	r.pipeline = pipeline
	r.ctx = pipeline.Context()
	log.Info("MetricsSender initializing")
}

func (r *MetricsSender) Close() {
	log.Info("MetricsSender is wrapping up")
	r.started.Store(false)
	r.readyToSend.Store(false)
}

func (r *MetricsSender) Info() *core.Info {
	return core.NewInfo("MetricsSender", "v0.0.1")
}

func (r *MetricsSender) Process(msg *core.Message) {
	if msg.Exact(core.RegistrationCompletedTopic) {
		r.readyToSend.Toggle()
		return
	}

	if msg.Exact(core.CommMetrics) {
		payloads, ok := msg.Data().([]core.Payload)
		if !ok {
			log.Warnf("Failed to coerce Message to []Payload: %v", msg.Data())
			return
		}
		for _, p := range payloads {
			if !r.readyToSend.Load() {
				continue
			}

			switch report := p.(type) {
			case *proto.MetricsReport:
				message := client.MessageFromMetrics(report)
				err := r.reporter.Send(r.ctx, message)

				if err != nil {
					log.Errorf("Failed to send MetricsReport: %v", err)
				}
			case *models.EventReport:
				err := r.reporter.Send(r.ctx, client.MessageFromEvents(report))
				if err != nil {
					l := len(report.Events)
					var sb strings.Builder
					for i := 0; i < l-1; i++ {
						sb.WriteString(report.Events[i].GetSecurityViolationEvent().SupportID)
						sb.WriteString(", ")
					}
					sb.WriteString(report.Events[l-1].GetSecurityViolationEvent().SupportID)
					log.Errorf("Failed to send EventReport with error: %v, supportID list: %s", err, sb.String())
				}
			}
		}
	} else if msg.Exact(core.AgentConfig) {
		cmd, ok := msg.Data().(*proto.Command_AgentConfig)
		if !ok {
			log.Warnf("Failed to coerce Message to *proto.Command_AgentConfig: %v", msg.Data())
			return
		}
		r.metricSenderBackoff(cmd)
	}
}
func (r *MetricsSender) metricSenderBackoff(cmd *proto.Command_AgentConfig) {
	agentConfig := cmd.AgentConfig

	if agentConfig == nil {
		log.Warnf("cannot update metric reporter client backoff settings with agent config nil, for a pause command %+v", cmd)
		return
	}
	if agentConfig.GetDetails() == nil {
		log.Warnf("cannot update metric reporter client backoff settings with agent details nil, for a pause command %+v", cmd)
		return
	}
	if agentConfig.GetDetails().GetServer() == nil {
		log.Warnf("cannot update metric reporter client backoff settings with server nil, for a pause command %+v", cmd)
		return
	}
	backoff := agentConfig.GetDetails().GetServer().Backoff
	if backoff == nil {
		log.Warnf("cannot update metric reporter client backoff settings with backoff settings nil, for a pause command %+v", cmd)
		return
	}

	smultiplier := sdk.BACKOFF_MULTIPLIER
	if backoff.GetMultiplier() != 0 {
		smultiplier = backoff.GetMultiplier()
	}

	jitter := sdk.BACKOFF_JITTER
	if backoff.GetRandomizationFactor() != 0 {
		jitter = backoff.GetRandomizationFactor()
	}

	cBackoff := sdk.BackoffSettings{
		InitialInterval: time.Duration(backoff.InitialInterval),
		MaxInterval:     time.Duration(backoff.MaxInterval),
		MaxElapsedTime:  time.Duration(backoff.MaxElapsedTime),
		Multiplier:      smultiplier,
		Jitter:          jitter,
	}
	log.Infof("update metric reporter client backoff settings to %+v, for a pause command %+v", cBackoff, cmd)
	r.reporter.WithBackoffSettings(cBackoff)
}

func (r *MetricsSender) Subscriptions() []string {
	return []string{core.CommMetrics, core.RegistrationCompletedTopic, core.AgentConfig}
}
