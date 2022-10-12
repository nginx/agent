package plugins

import (
	"context"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

type DataPlaneUpdate struct {
	messagePipeline core.MessagePipeInterface
	ctx             context.Context
	updateTicker    *time.Ticker
	env             core.Environment
	dataplaneUpdate *proto.DataplaneUpdate
	meta            *proto.Metadata
}

func NewDataPlaneUpdate(config *config.Config, env core.Environment, meta *proto.Metadata) *DataPlaneUpdate {
	log.Tracef("Dataplane update interval %s", config.Dataplane.Status.PollInterval)

	return &DataPlaneUpdate{
		meta:         meta,
		updateTicker: time.NewTicker(getPollIntervalFrom(config)),
	}
}

func (dpu *DataPlaneUpdate) Init(pipeline core.MessagePipeInterface) {
	log.Info("DataPlaneUpdate initializing")
	dpu.messagePipeline = pipeline
	dpu.ctx = dpu.messagePipeline.Context()

	quit := make(chan struct{})

	// this should call ticker.Stop()
	defer close(quit)

	go func() {
		for {
			select {
			case <-dpu.updateTicker.C:
				// do stuff
				dpu.sendDataplaneUpdate()
			case <-quit:
				dpu.updateTicker.Stop()
				return
			}
		}
	}()
}

func (dpu *DataPlaneUpdate) Close() {
	log.Info("DataPlaneUpdate is wrapping up")
	dpu.updateTicker.Stop()
}

func (dpu *DataPlaneUpdate) Info() *core.Info {
	return core.NewInfo("DataPlaneUpdate", "v0.0.1")
}

func (dpu *DataPlaneUpdate) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.AgentConfigChanged):
		// If the agent config on disk changed update DataPlaneStatus with relevant config info
		conf, err := config.GetConfig(dpu.env.GetSystemUUID())
		if err != nil {
			log.Errorf("Failed to load config for updating: %v", err)
			return
		}
		log.Debugf("DataPlaneStatus is updating to a new config - %v", conf)

		dpu.updateTicker.Reset(getPollIntervalFrom(conf))

		// move this to software details?
		// dps.configDirs = conf.ConfigDirs

	case msg.Exact(core.NginxAppProtectDetailsGenerated):
		switch commandData := msg.Data().(type) {
		case *proto.DataplaneSoftwareDetails_AppProtectWafDetails:
			log.Debugf("DataPlaneStatus is syncing with NAP details - %+v", commandData.AppProtectWafDetails)
			// dpu.napDetails = commandData
		default:
			log.Errorf("Expected the type %T but got %T", &proto.DataplaneSoftwareDetails_AppProtectWafDetails{}, commandData)
		}
	}
}

func (dpu *DataPlaneUpdate) Subscriptions() []string {
	return []string{}
}

func (dpu *DataPlaneUpdate) dataplaneUpdateMessage() *proto.DataplaneUpdate {
	processes := dpu.env.Processes()
	log.Tracef("dataplaneStatus: processes %v", processes)
    update := &proto.DataplaneUpdate{
    	Host:                     &proto.HostInfo{},
    	DataplaneSoftwareDetails: []*proto.DataplaneSoftwareDetails{},
	}

	if (!cmp.Equal(dpu.dataplaneUpdate, update)) {
		dpu.dataplaneUpdate = update
	}

	return dpu.dataplaneUpdate
}

func (dpu *DataPlaneUpdate) sendDataplaneUpdate(pipeline core.MessagePipeInterface) {
	meta := *dpu.meta
	meta.MessageId = uuid.New().String()
	statusData := proto.Command_DataplaneUpdate{
		DataplaneUpdate: dpu.dataplaneUpdateMessage(),
	}
	log.Tracef("sendDataplaneStatus statusData: %v", statusData)
	pipeline.Process(
		core.NewMessage(core.CommStatus, &proto.Command{
			Meta: &meta,
			Data: &statusData,
		}),
	)
}
