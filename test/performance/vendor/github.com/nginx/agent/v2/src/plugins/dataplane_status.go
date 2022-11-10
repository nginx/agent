package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

type DataPlaneStatus struct {
	messagePipeline       core.MessagePipeInterface
	ctx                   context.Context
	sendStatus            chan bool
	healthTicker          *time.Ticker
	interval              time.Duration
	meta                  *proto.Metadata
	binary                core.NginxBinary
	env                   core.Environment
	version               string
	tags                  *[]string
	configDirs            string
	lastSendDetails       time.Time
	envHostInfo           *proto.HostInfo
	statusUrls            map[string]string
	reportInterval        time.Duration
	napDetails            *proto.DataplaneSoftwareDetails_AppProtectWafDetails
	agentActivityStatuses []*proto.AgentActivityStatus
	napDetailsMutex       sync.RWMutex
}

const (
	defaultMinInterval = time.Second * 30
)

func NewDataPlaneStatus(config *config.Config, meta *proto.Metadata, binary core.NginxBinary, env core.Environment, version string) *DataPlaneStatus {
	log.Tracef("Dataplane status interval %s", config.Dataplane.Status.PollInterval)
	pollInt := config.Dataplane.Status.PollInterval
	if pollInt < defaultMinInterval {
		pollInt = defaultMinInterval
		log.Warnf("interval set to %s, provided value (%s) less than minimum", pollInt, config.Dataplane.Status.PollInterval)
	}
	return &DataPlaneStatus{
		sendStatus:      make(chan bool),
		healthTicker:    time.NewTicker(pollInt),
		interval:        pollInt,
		meta:            meta,
		binary:          binary,
		env:             env,
		version:         version,
		tags:            &config.Tags,
		configDirs:      config.ConfigDirs,
		statusUrls:      make(map[string]string),
		reportInterval:  config.Dataplane.Status.ReportInterval,
		napDetailsMutex: sync.RWMutex{},
		// Intentionally empty as it will be set later
		napDetails: nil,
	}
}

func (dps *DataPlaneStatus) Init(pipeline core.MessagePipeInterface) {
	log.Info("DataPlaneStatus initializing")
	dps.messagePipeline = pipeline
	dps.ctx = dps.messagePipeline.Context()
	dps.healthGoRoutine(pipeline)
}

func (dps *DataPlaneStatus) Close() {
	log.Info("DataPlaneStatus is wrapping up")
	dps.healthTicker.Stop()
	dps.sendStatus <- true
}

func (dps *DataPlaneStatus) Info() *core.Info {
	return core.NewInfo("DataPlaneStatus", "v0.0.2")
}

func (dps *DataPlaneStatus) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.AgentConfigChanged):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		// If the agent config on disk changed update DataPlaneStatus with relevant config info
		dps.syncAgentConfigChange()

	case msg.Exact(core.DataplaneSoftwareDetailsUpdated):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		switch data := msg.Data().(type) {
		case *proto.DataplaneSoftwareDetails_AppProtectWafDetails:
			log.Debugf("DataplaneStatus is syncing with NAP details - %+v", data.AppProtectWafDetails)
			dps.napDetailsMutex.Lock()
			dps.napDetails = data
			dps.napDetailsMutex.Unlock()
		}

	case msg.Exact(core.NginxConfigValidationPending):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		switch data := msg.Data().(type) {
		case *proto.AgentActivityStatus:
			dps.updateAgentActivityStatuses(data)
		default:
			log.Errorf("Expected the type %T but got %T", &proto.AgentActivityStatus{}, data)
		}

	case msg.Exact(core.NginxConfigApplyFailed) || msg.Exact(core.NginxConfigApplySucceeded):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		switch data := msg.Data().(type) {
		case *proto.AgentActivityStatus:
			dps.updateAgentActivityStatuses(data)
			dps.sendDataplaneStatus(dps.messagePipeline, false)
			dps.removeAgentActivityStatus(data)
		default:
			log.Errorf("Expected the type %T but got %T", &proto.AgentActivityStatus{}, data)
		}
	}
}

func (dps *DataPlaneStatus) Subscriptions() []string {
	return []string{
		core.AgentConfigChanged,
		core.DataplaneSoftwareDetailsUpdated,
		core.NginxConfigValidationPending,
		core.NginxConfigApplyFailed,
		core.NginxConfigApplySucceeded,
	}
}

func (dps *DataPlaneStatus) updateAgentActivityStatuses(newAgentActivityStatus *proto.AgentActivityStatus) {
	log.Tracef("DataplaneStatus: Adding %v to agentActivityStatuses", newAgentActivityStatus)
	if _, ok := newAgentActivityStatus.GetStatus().(*proto.AgentActivityStatus_NginxConfigStatus); ok {
		foundExistingNginxStatus := false
		for index, agentActivityStatus := range dps.agentActivityStatuses {
			if _, ok := agentActivityStatus.GetStatus().(*proto.AgentActivityStatus_NginxConfigStatus); ok {
				dps.agentActivityStatuses[index] = newAgentActivityStatus
				log.Tracef("DataplaneStatus: Updated agentActivityStatus with new status %v", newAgentActivityStatus)
				foundExistingNginxStatus = true
			}
		}

		if !foundExistingNginxStatus {
			dps.agentActivityStatuses = append(dps.agentActivityStatuses, newAgentActivityStatus)
			log.Tracef("DataplaneStatus: Added new status %v to agentActivityStatus", newAgentActivityStatus)
		}
	}
}

func (dps *DataPlaneStatus) removeAgentActivityStatus(agentActivityStatus *proto.AgentActivityStatus) {
	log.Tracef("DataplaneStatus: Removing %v from agentActivityStatuses", agentActivityStatus)
	if _, ok := agentActivityStatus.GetStatus().(*proto.AgentActivityStatus_NginxConfigStatus); ok {
		for index, agentActivityStatus := range dps.agentActivityStatuses {
			if _, ok := agentActivityStatus.GetStatus().(*proto.AgentActivityStatus_NginxConfigStatus); ok {
				dps.agentActivityStatuses = append(dps.agentActivityStatuses[:index], dps.agentActivityStatuses[index+1:]...)
				log.Tracef("DataplaneStatus: Removed %v from agentActivityStatus", agentActivityStatus)
			}
		}
	}
}

func (dps *DataPlaneStatus) sendDataplaneStatus(pipeline core.MessagePipeInterface, forceDetails bool) {
	meta := *dps.meta
	meta.MessageId = uuid.New().String()
	statusData := proto.Command_DataplaneStatus{
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

func (dps *DataPlaneStatus) healthGoRoutine(pipeline core.MessagePipeInterface) {
	dps.sendDataplaneStatus(pipeline, true)
	go func() {
		for {
			select {
			case <-dps.sendStatus:
				return
			case t := <-dps.healthTicker.C:
				log.Tracef("healthGoRoutine Status woke up at %v", t)
				dps.sendDataplaneStatus(pipeline, false)
			}
		}
	}()
}

func (dps *DataPlaneStatus) dataplaneStatus(forceDetails bool) *proto.DataplaneStatus {
	processes := dps.env.Processes()
	log.Tracef("dataplaneStatus: processes %v", processes)
	forceDetails = forceDetails || time.Now().UTC().Add(-dps.reportInterval).After(dps.lastSendDetails)
	return &proto.DataplaneStatus{
		Host:                     dps.hostInfo(forceDetails),
		Details:                  dps.detailsForProcess(processes, forceDetails),
		Healths:                  dps.healthForProcess(processes),
		DataplaneSoftwareDetails: dps.dataplaneSoftwareDetails(),
		AgentActivityStatus:      dps.agentActivityStatuses,
	}
}

func (dps *DataPlaneStatus) dataplaneSoftwareDetails() []*proto.DataplaneSoftwareDetails {
	allDetails := make([]*proto.DataplaneSoftwareDetails, 0)

	dps.napDetailsMutex.RLock()
	defer dps.napDetailsMutex.RUnlock()
	if dps.napDetails != nil {
		napDetails := &proto.DataplaneSoftwareDetails{
			Data: dps.napDetails,
		}
		allDetails = append(allDetails, napDetails)
	}

	return allDetails
}

func (dps *DataPlaneStatus) hostInfo(send bool) (info *proto.HostInfo) {
	// this sets send if we are forcing details, or it has been 24 hours since the last send
	hostInfo := dps.env.NewHostInfo(dps.version, dps.tags, dps.configDirs, true)
	if !send && cmp.Equal(dps.envHostInfo, hostInfo) {
		return nil
	}

	dps.envHostInfo = hostInfo
	log.Tracef("hostInfo: %v", hostInfo)

	return dps.envHostInfo
}

func (dps *DataPlaneStatus) detailsForProcess(processes []core.Process, send bool) (details []*proto.NginxDetails) {
	log.Tracef("detailsForProcess processes: %v", processes)

	nowUTC := time.Now().UTC()
	statusAPIUpdated := false
	// this sets send if we are forcing details, or it has been 24 hours since the last send
	for _, p := range processes {
		if !p.IsMaster {
			continue
		}
		detail := dps.binary.GetNginxDetailsFromProcess(p)
		if dps.statusUrls[detail.NginxId] != detail.StatusUrl {
			log.Infof("NGINX status API updated.  Old status API: %v, new status API: %v", dps.statusUrls[detail.NginxId], detail.StatusUrl)
			dps.statusUrls[detail.NginxId] = detail.StatusUrl
			statusAPIUpdated = true
		}
		details = append(details, detail)
		// spec says process CreateTime is unix UTC in MS
		if time.Unix(p.CreateTime/1000, 0).After(dps.lastSendDetails) {
			// set send because this process has started since the last send
			send = true
		}
	}

	// If the statusAPI was updated send a new NGINX status API updated message
	if statusAPIUpdated {
		dps.messagePipeline.Process(core.NewMessage(core.NginxStatusAPIUpdate, ""))
	}

	if !send {
		return nil
	}

	dps.lastSendDetails = nowUTC

	return details
}

func (dps *DataPlaneStatus) healthForProcess(processes []core.Process) (healths []*proto.NginxHealth) {
	heathDetails := make(map[string]*proto.NginxHealth)
	instanceProcessCount := make(map[string]int)
	log.Tracef("healthForProcess processes: %v", processes)

	for _, p := range processes {
		instanceID := dps.binary.GetNginxIDForProcess(p)
		log.Tracef("Process: %v instanceID %s", p, instanceID)
		if _, ok := heathDetails[instanceID]; !ok {
			heathDetails[instanceID] = &proto.NginxHealth{
				NginxId:     instanceID,
				NginxStatus: proto.NginxHealth_ACTIVE,
			}
			instanceProcessCount[instanceID] = 0
		}
		instanceProcessCount[instanceID]++
		log.Tracef("IsRunning: %t Status: %s", p.IsRunning, p.Status)
		if !p.IsRunning {
			reason := fmt.Sprintf("NginxID: %s pid: %d is degraded: %s", instanceID, p.Pid, p.Status)
			if heathDetails[instanceID].NginxStatus == proto.NginxHealth_DEGRADED {
				reason = fmt.Sprintf("%s\n%s", reason, heathDetails[instanceID].DegradedReason)
			}
			heathDetails[instanceID].DegradedReason = reason
			heathDetails[instanceID].NginxStatus = proto.NginxHealth_DEGRADED
		}
	}

	for instanceID, health := range heathDetails {
		log.Tracef("instanceID: %s health: %s", instanceID, health)

		if instanceProcessCount[instanceID] <= 1 {
			reason := "does not have enough children"
			if heathDetails[instanceID].NginxStatus == proto.NginxHealth_DEGRADED {
				reason = fmt.Sprintf("%s\n%s", reason, heathDetails[instanceID].DegradedReason)
			}
			heathDetails[instanceID].DegradedReason = reason
			health.NginxStatus = proto.NginxHealth_DEGRADED
		}
		healths = append(healths, health)
	}
	return healths
}

func (dps *DataPlaneStatus) syncAgentConfigChange() {
	conf, err := config.GetConfig(dps.env.GetSystemUUID())
	if err != nil {
		log.Errorf("Failed to load config for updating: %v", err)
		return
	}
	log.Debugf("DataPlaneStatus is updating to a new config - %v", conf)

	pollInt := conf.Dataplane.Status.PollInterval
	if pollInt < defaultMinInterval {
		pollInt = defaultMinInterval
		log.Warnf("interval set to %s, provided value (%s) less than minimum", pollInt, conf.Dataplane.Status.PollInterval)
	}

	if conf.DisplayName == "" {
		conf.DisplayName = dps.env.GetHostname()
		log.Infof("setting displayName to %s", conf.DisplayName)
	}

	// Update DataPlaneStatus with relevant config info
	dps.interval = pollInt
	dps.tags = &conf.Tags
	dps.configDirs = conf.ConfigDirs
}
