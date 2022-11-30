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

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"

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
					log.Errorf("Failed to send MetricsReport: %v, data: %+v", err, report)
				} else {
					log.Tracef("MetricsReport sent, %v", report)
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
	}
}

func (r *MetricsSender) Subscriptions() []string {
	return []string{core.CommMetrics, core.RegistrationCompletedTopic}
}
