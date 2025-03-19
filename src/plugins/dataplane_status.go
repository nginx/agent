/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */
package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
	log "github.com/sirupsen/logrus"
)

type DataPlaneStatus struct {
	messagePipeline                  core.MessagePipeInterface
	ctx                              context.Context
	sendStatus                       chan bool
	healthTicker                     *time.Ticker
	interval                         time.Duration
	meta                             *proto.Metadata
	binary                           core.NginxBinary
	env                              core.Environment
	version                          string
	tags                             *[]string
	configDirs                       string
	lastSendDetails                  time.Time
	envHostInfo                      *proto.HostInfo
	reportInterval                   time.Duration
	softwareDetails                  map[string]*proto.DataplaneSoftwareDetails
	nginxConfigActivityStatuses      map[string]*proto.AgentActivityStatus
	nginxConfigActivityStatusesMutex sync.RWMutex
	softwareDetailsMutex             sync.RWMutex
	structMu                         sync.RWMutex
	processes                        []*core.Process
}

const (
	defaultMinInterval = time.Second * 30
)

func NewDataPlaneStatus(config *config.Config, meta *proto.Metadata, binary core.NginxBinary, env core.Environment, processes []*core.Process) *DataPlaneStatus {
	log.Tracef("Dataplane status interval %s", config.Dataplane.Status.PollInterval)
	pollInt := config.Dataplane.Status.PollInterval
	if pollInt < defaultMinInterval {
		pollInt = defaultMinInterval
		log.Warnf("interval set to %s, provided value (%s) less than minimum", pollInt, config.Dataplane.Status.PollInterval)
	}
	return &DataPlaneStatus{
		sendStatus:                  make(chan bool),
		healthTicker:                time.NewTicker(pollInt),
		interval:                    pollInt,
		meta:                        meta,
		binary:                      binary,
		env:                         env,
		version:                     config.Version,
		tags:                        &config.Tags,
		configDirs:                  config.ConfigDirs,
		reportInterval:              config.Dataplane.Status.ReportInterval,
		nginxConfigActivityStatuses: make(map[string]*proto.AgentActivityStatus),
		softwareDetails:             make(map[string]*proto.DataplaneSoftwareDetails),
		processes:                   processes,
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
	dps.nginxConfigActivityStatusesMutex.Lock()
	dps.nginxConfigActivityStatuses = nil
	dps.nginxConfigActivityStatusesMutex.Unlock()
	dps.softwareDetailsMutex.Lock()
	dps.softwareDetails = nil
	dps.softwareDetailsMutex.Unlock()
	dps.healthTicker.Stop()
	dps.sendStatus <- true
}

func (dps *DataPlaneStatus) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureDataPlaneStatus, "v0.0.2")
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
		case *payloads.DataplaneSoftwareDetailsUpdate:
			dps.softwareDetailsMutex.Lock()
			dps.softwareDetails[data.GetPluginName()] = data.GetDataplaneSoftwareDetails()
			dps.softwareDetailsMutex.Unlock()
		}
	case msg.Exact(core.NginxConfigValidationPending):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		switch data := msg.Data().(type) {
		case *proto.AgentActivityStatus:
			dps.updateNginxConfigActivityStatuses(data)
		default:
			log.Errorf("Expected the type %T but got %T", &proto.AgentActivityStatus{}, data)
		}
	case msg.Exact(core.NginxConfigApplyFailed) || msg.Exact(core.NginxConfigApplySucceeded):
		log.Tracef("DataplaneStatus: %T message from topic %s received", msg.Data(), msg.Topic())
		switch data := msg.Data().(type) {
		case *proto.AgentActivityStatus:
			dps.updateNginxConfigActivityStatuses(data)
			dps.sendDataplaneStatus(dps.messagePipeline, false)
		default:
			log.Errorf("Expected the type %T but got %T", &proto.AgentActivityStatus{}, data)
		}
	case msg.Exact(core.NginxDetailProcUpdate):
		dps.structMu.Lock()
		dps.processes = msg.Data().([]*core.Process)
		dps.structMu.Unlock()
	}
}

func (dps *DataPlaneStatus) Subscriptions() []string {
	return []string{
		core.AgentConfigChanged,
		core.DataplaneSoftwareDetailsUpdated,
		core.NginxConfigValidationPending,
		core.NginxConfigApplyFailed,
		core.NginxConfigApplySucceeded,
		core.NginxDetailProcUpdate,
	}
}

func (dps *DataPlaneStatus) updateNginxConfigActivityStatuses(newAgentActivityStatus *proto.AgentActivityStatus) {
	log.Tracef("DataplaneStatus: Updating nginxConfigActivityStatuses with %v", newAgentActivityStatus)
	if _, ok := newAgentActivityStatus.GetStatus().(*proto.AgentActivityStatus_NginxConfigStatus); dps.nginxConfigActivityStatuses != nil && ok {
		dps.nginxConfigActivityStatusesMutex.Lock()
		dps.nginxConfigActivityStatuses[newAgentActivityStatus.GetNginxConfigStatus().GetNginxId()] = newAgentActivityStatus
		dps.nginxConfigActivityStatusesMutex.Unlock()
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
			case <-dps.ctx.Done():
				return
			}
		}
	}()
}

func (dps *DataPlaneStatus) dataplaneStatus(forceDetails bool) *proto.DataplaneStatus {
	forceDetails = forceDetails || time.Now().UTC().Add(-dps.reportInterval).After(dps.lastSendDetails)

	dps.nginxConfigActivityStatusesMutex.Lock()
	defer dps.nginxConfigActivityStatusesMutex.Unlock()
	agentActivityStatuses := []*proto.AgentActivityStatus{}
	for _, nginxConfigActivityStatus := range dps.nginxConfigActivityStatuses {
		agentActivityStatuses = append(agentActivityStatuses, nginxConfigActivityStatus)
	}
	dps.softwareDetailsMutex.Lock()
	defer dps.softwareDetailsMutex.Unlock()
	dataplaneSoftwareDetails := []*proto.DataplaneSoftwareDetails{}
	for _, softwareDetail := range dps.softwareDetails {
		dataplaneSoftwareDetails = append(dataplaneSoftwareDetails, softwareDetail)
	}
	dataplaneStatus := &proto.DataplaneStatus{
		Host:                     dps.hostInfo(forceDetails),
		Details:                  dps.detailsForProcess(dps.processes, forceDetails),
		Healths:                  dps.healthForProcess(dps.processes),
		DataplaneSoftwareDetails: dataplaneSoftwareDetails,
		AgentActivityStatus:      agentActivityStatuses,
	}
	return dataplaneStatus
}

func (dps *DataPlaneStatus) hostInfo(send bool) (info *proto.HostInfo) {
	// this sets send if we are forcing details, or it has been 24 hours since the last send
	dps.structMu.Lock()
	defer dps.structMu.Unlock()
	hostInfo := dps.env.NewHostInfo(dps.version, dps.tags, dps.configDirs, send)
	if !send && cmp.Equal(dps.envHostInfo, hostInfo) {
		return nil
	}

	dps.envHostInfo = hostInfo
	log.Tracef("hostInfo: %v", hostInfo)

	return hostInfo
}

func (dps *DataPlaneStatus) detailsForProcess(processes []*core.Process, send bool) (details []*proto.NginxDetails) {
	log.Tracef("detailsForProcess processes: %v", processes)
	nowUTC := time.Now().UTC()
	// this sets send if we are forcing details, or it has been 24 hours since the last send
	for _, p := range processes {
		if !p.IsMaster {
			continue
		}
		details = append(details, dps.binary.GetNginxDetailsFromProcess(p))
		// spec says process CreateTime is unix UTC in MS
		if time.UnixMilli(p.CreateTime).After(dps.lastSendDetails) {
			// set send because this process has started since the last send
			send = true
		}
	}

	if !send {
		return nil
	}

	dps.structMu.Lock()
	dps.lastSendDetails = nowUTC
	dps.structMu.Unlock()

	return details
}

func (dps *DataPlaneStatus) healthForProcess(processes []*core.Process) (healths []*proto.NginxHealth) {
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
	dps.structMu.Lock()

	dps.interval = pollInt
	dps.tags = &conf.Tags
	dps.configDirs = conf.ConfigDirs

	dps.structMu.Unlock()
}
