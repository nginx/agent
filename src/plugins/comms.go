package plugins

import (
	"context"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
)

const (
	DefaultMetricsChanLength = 4 * 1024
	DefaultEventsChanLength  = 4 * 1024
)

type Comms struct {
	reporter         client.MetricReporter
	pipeline         core.MessagePipeInterface
	reportChan       chan *proto.MetricsReport
	reportEventsChan chan *models.EventReport
	ctx              context.Context
	started          *atomic.Bool
	readyToSend      *atomic.Bool
	wait             sync.WaitGroup
}

func NewComms(reporter client.MetricReporter) *Comms {
	return &Comms{
		reporter:         reporter,
		reportChan:       make(chan *proto.MetricsReport, DefaultMetricsChanLength),
		reportEventsChan: make(chan *models.EventReport, DefaultEventsChanLength),
		started:          atomic.NewBool(false),
		readyToSend:      atomic.NewBool(false),
		wait:             sync.WaitGroup{},
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
	go r.reportLoop()
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
			switch report := p.(type) {
			case *proto.MetricsReport:
				select {
				case <-r.ctx.Done():
					err := r.ctx.Err()
					if err != nil {
						log.Errorf("error in done context Process in comms %v", err)
					}
					return
				case r.reportChan <- report:
					// report queued
					log.Debug("metrics report queued")
				}
			case *models.EventReport:
				select {
				case <-r.ctx.Done():
					err := r.ctx.Err()
					if err != nil {
						log.Errorf("error in done context Process in comms %v", err)
					}
					return
				case r.reportEventsChan <- report:
					// report queued
					log.Debug("events report queued")
				}
			}
		}
	}
}

func (r *Comms) Subscriptions() []string {
	return []string{core.CommMetrics, core.RegistrationCompletedTopic}
}

func (r *Comms) reportLoop() {
	r.wait.Add(1)
	defer r.wait.Done()
	for {
		if !r.readyToSend.Load() {
			continue
		}
		select {
		case <-r.ctx.Done():
			err := r.ctx.Err()
			if err != nil {
				log.Errorf("error in done context reportLoop %v", err)
			}
			log.Debug("reporter loop exiting")
			return
		case report := <-r.reportChan:
			err := r.reporter.Send(r.ctx, client.MessageFromMetrics(report))
			if err != nil {
				log.Errorf("Failed to send MetricsReport: %v, data: %+v", err, report)
			} else {
				log.Tracef("MetricsReport sent, %v", report)
			}
		case report := <-r.reportEventsChan:
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
			} else {
				log.Tracef("EventReport sent, %v", report)
			}
		}
	}
}
