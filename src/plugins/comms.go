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

type Comms struct {
	reporter    client.MetricReporter
	pipeline    core.MessagePipeInterface
	ctx         context.Context
	started     *atomic.Bool
	readyToSend *atomic.Bool
}

func NewComms(reporter client.MetricReporter) *Comms {
	return &Comms{
		reporter:    reporter,
		started:     atomic.NewBool(false),
		readyToSend: atomic.NewBool(false),
	}
}

func (r *Comms) Init(pipeline core.MessagePipeInterface) {
	if r.started.Load() {
		return
	}
	r.started.Toggle()
	r.pipeline = pipeline
	r.ctx = pipeline.Context()
	log.Info("Comms initializing")
}

func (r *Comms) Close() {
	log.Info("Comms is wrapping up")
	r.started.Store(false)
	r.readyToSend.Store(false)
}

func (r *Comms) Info() *core.Info {
	return core.NewInfo("Communications", "v0.0.2")
}

func (r *Comms) Process(msg *core.Message) {
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

func (r *Comms) Subscriptions() []string {
	return []string{core.CommMetrics, core.RegistrationCompletedTopic}
}
