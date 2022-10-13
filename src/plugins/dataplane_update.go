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
	binary          core.NginxBinary
	dataplaneUpdate *proto.DataplaneUpdate
	meta            *proto.Metadata
	config          *config.Config
	version         string
	napDetails      *proto.DataplaneSoftwareDetails_AppProtectWafDetails
}

func NewDataPlaneUpdate(config *config.Config, binary core.NginxBinary, env core.Environment, meta *proto.Metadata, version string) *DataPlaneUpdate {
	log.Tracef("Dataplane update interval %s", config.Dataplane.Status.PollInterval)

	return &DataPlaneUpdate{
		updateTicker: time.NewTicker(getPollIntervalFrom(config)),
		env:          env,
		binary:       binary,
		meta:         meta,
		config:	      config,
		version:      version,
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
		dpu.syncAgentConfigChange()
	case msg.Exact(core.NginxAppProtectDetailsGenerated):
		dpu.napDetails = getNAPDetails(msg)
	}
}

func (dpu *DataPlaneUpdate) syncAgentConfigChange() {
	conf, err := config.GetConfig(dpu.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	log.Debugf("DataPlaneStatus is updating to a new config - %v", conf)

	dpu.updateTicker.Reset(getPollIntervalFrom(conf))

	if conf.DisplayName == "" {
		conf.DisplayName = dpu.env.GetHostname()
		log.Infof("setting displayName to %s", conf.DisplayName)
	}

	dpu.config = conf
}

func (dpu *DataPlaneUpdate) Subscriptions() []string {
	return []string{core.AgentConfigChanged, core.NginxAppProtectDetailsGenerated}
}

func (dpu *DataPlaneUpdate) sendDataplaneUpdate() {
	meta := *dpu.meta
	meta.MessageId = uuid.New().String()

    update := &proto.DataplaneUpdate{
    	Host:                     dpu.getHostInfo(),
    	DataplaneSoftwareDetails: getSoftwareDetails(dpu.env, dpu.binary, dpu.napDetails),
	}

	if (!cmp.Equal(dpu.dataplaneUpdate, update)) {
		statusData := proto.Command_DataplaneUpdate{
			DataplaneUpdate: update,
		}
		log.Tracef("sendDataplaneStatus statusData: %v", statusData)
		dpu.dataplaneUpdate = update
		dpu.messagePipeline.Process(
			core.NewMessage(core.CommStatus, &proto.Command{
				Meta: &meta,
				Data: &statusData,
			}),
		)
	}
}

func (dpu *DataPlaneUpdate) getHostInfo() *proto.HostInfo {
	hostInfo := dpu.env.NewHostInfo(dpu.version, &dpu.config.Tags, dpu.config.ConfigDirs, true)
	log.Tracef("hostInfo: %v", hostInfo)
	return hostInfo
}

func getSoftwareDetails(env core.Environment, binary core.NginxBinary, napDetails *proto.DataplaneSoftwareDetails_AppProtectWafDetails) (details []*proto.DataplaneSoftwareDetails) {
	processes := env.Processes()
	log.Tracef("dataplane update: processes %v", processes)
	for _, p := range processes {
		if !p.IsMaster {
			continue
		}
		detail := binary.GetNginxDetailsFromProcess(p)

		data := &proto.DataplaneSoftwareDetails{
			Data: &proto.DataplaneSoftwareDetails_NginxDetails{
				NginxDetails: detail,
			},
		}
		details = append(details, data)
	}

	// add nap details
	details = append(details, &proto.DataplaneSoftwareDetails{
		Data: napDetails,
	})
	
	log.Tracef("software details: %v", details)
	return details
}
