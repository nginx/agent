package plugins

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

type DataPlaneUpdate struct {
	messagePipeline core.MessagePipeInterface
	ctx             context.Context
	updateTicker    *time.Ticker
	reportInterval  time.Duration
	env             core.Environment
	dataplaneUpdate *proto.DataplaneUpdate
}

func NewDataPlaneUpdate(config *config.Config, env core.Environment) *DataPlaneUpdate {
	log.Tracef("Dataplane status interval %s", config.Dataplane.Status.PollInterval)
	pollInt := config.Dataplane.Status.PollInterval
	if pollInt < defaultMinInterval {
		pollInt = defaultMinInterval
		log.Warnf("interval set to %s, provided value (%s) less than minimum", pollInt, config.Dataplane.Status.PollInterval)
	}
	
	return &DataPlaneUpdate{
		updateTicker:   time.NewTicker(pollInt),
		reportInterval: config.Dataplane.Status.ReportInterval,
	}
}

func (dpu *DataPlaneUpdate) Init(pipeline core.MessagePipeInterface) {
	log.Info("DataPlaneUpdate initializing")
	dpu.messagePipeline = pipeline
	dpu.ctx = dpu.messagePipeline.Context()

	done := make(chan bool)
    ticker := time.NewTicker(1 * time.Second)
    for _ = range ticker.C {
        fmt.Println("Tock")
    }

    go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Tick at", t)
			}
		}
	}()

    time.Sleep(100 * time.Second)
	ticker.Stop()
	done <- true
	fmt.Println("Ticker stopped")
}

func (dpu *DataPlaneUpdate) Close() {
	log.Info("DataPlaneUpdate is wrapping up")
	dpu.updateTicker.Stop()
}

func (dps *DataPlaneUpdate) Info() *core.Info {
	return core.NewInfo("DataPlaneUpdate", "v0.0.1")
}

func (dps *DataPlaneUpdate) Process(msg *core.Message) {
}

func (dpu *DataPlaneUpdate) Subscriptions() []string {
	return []string{}
}

func (dpu *DataPlaneUpdate) dataplaneUpdateMessage(forceDetails bool) *proto.DataplaneUpdate {
	processes := dpu.env.Processes()
	log.Tracef("dataplaneStatus: processes %v", processes)
	forceDetails = forceDetails || time.Now().UTC().Add(-dpu.reportInterval).After(dpu.lastSendDetails)
	return dpu.dataplaneUpdate
}

func (dps *DataPlaneStatus) sendDataplaneUpdate(pipeline core.MessagePipeInterface, forceDetails bool) {
	meta := *dps.meta
	meta.MessageId = uuid.New().String()
	statusData := proto.Comm{
		DataplaneStatus: dps.dataplaneStatus(forceDetails),
	}
	log.Tracef("sendDataplaneStatus statusData: %v", statusData)
	pipeline.Process(
		core.NewMessage(core.CommStatus, &proto.Command{
			Meta: &meta,
			Data: &statusData,
		}),
	)
}
