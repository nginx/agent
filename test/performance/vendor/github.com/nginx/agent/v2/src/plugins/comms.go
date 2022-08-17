package plugins

import (
	"context"
	"sync"

	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"
)

const (
	DefaultMetricsChanLength = 4 * 1024
)

type Comms struct {
	reporter    client.MetricReporter
	pipeline    core.MessagePipeInterface
	reportChan  chan *proto.MetricsReport
	ctx         context.Context
	started     *atomic.Bool
	readyToSend *atomic.Bool
	wait        sync.WaitGroup
}

func NewComms(reporter client.MetricReporter) *Comms {
	return &Comms{
		reporter:    reporter,
		reportChan:  make(chan *proto.MetricsReport, DefaultMetricsChanLength),
		started:     atomic.NewBool(false),
		readyToSend: atomic.NewBool(false),
		wait:        sync.WaitGroup{},
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
	r.wait.Done()
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
					log.Debug("report queued")
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
		}
	}
}
