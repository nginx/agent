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

	"github.com/nginx/agent/sdk/v2/backoff"
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
		switch cmd := msg.Data().(type) {
		case *proto.Command:
			r.metricSenderBackoff(cmd.GetAgentConfig())
		default:
			log.Warnf("metrics sender expected %T type, but got: %T", &proto.Command{}, msg.Data())
		}
	}
}

func (r *MetricsSender) metricSenderBackoff(agentConfig *proto.AgentConfig) {
	log.Debugf("metricSenderBackoff in metrics_sender.go with agent config %+v ", agentConfig)

	if agentConfig == nil {
		log.Warnf("update metric reporter client with default backoff settings as agent config nil, for a pause command %+v", agentConfig)
		r.reporter.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}
	if agentConfig.GetDetails() == nil {
		log.Warnf("update metric reporter client with default backoff settings as agent details nil, for a pause command %+v", agentConfig)
		r.reporter.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}
	if agentConfig.GetDetails().GetServer() == nil {
		log.Warnf("update metric reporter client with default backoff settings as server nil, for a pause command %+v", agentConfig)
		r.reporter.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}
	backoffSetting := agentConfig.GetDetails().GetServer().Backoff
	if backoffSetting == nil {
		log.Warnf("update metric reporter client with default backoff settings as backoff settings nil, for a pause command %+v", agentConfig)
		r.reporter.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}

	multiplier := backoff.BACKOFF_MULTIPLIER
	if backoffSetting.GetMultiplier() != 0 {
		multiplier = backoffSetting.GetMultiplier()
	}

	jitter := backoff.BACKOFF_JITTER
	if backoffSetting.GetRandomizationFactor() != 0 {
		jitter = backoffSetting.GetRandomizationFactor()
	}

	cBackoff := backoff.BackoffSettings{
		InitialInterval: time.Duration(backoffSetting.InitialInterval * int64(time.Second)),
		MaxInterval:     time.Duration(backoffSetting.MaxInterval * int64(time.Second)),
		MaxElapsedTime:  time.Duration(backoffSetting.MaxElapsedTime * int64(time.Second)),
		Multiplier:      multiplier,
		Jitter:          jitter,
	}
	log.Debugf("update metric reporter client backoff settings to %+v, for a pause command %+v", cBackoff, agentConfig)
	r.reporter.WithBackoffSettings(cBackoff)
}

func (r *MetricsSender) Subscriptions() []string {
	return []string{core.CommMetrics, core.RegistrationCompletedTopic, core.AgentConfig}
}
