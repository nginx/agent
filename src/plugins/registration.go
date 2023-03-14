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

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
)

const (
	// Timeout for registration attempting to gather DataplaneSoftwareDetails
	dataplaneSoftwareDetailsMaxWaitTime = time.Duration(5 * time.Second)
	// Time between attempts to gather DataplaneSoftwareDetails
	softwareDetailsOperationInterval = time.Duration(1 * time.Second)
)

type OneTimeRegistration struct {
	agentVersion                  string
	tags                          *[]string
	meta                          *proto.Metadata
	config                        *config.Config
	env                           core.Environment
	host                          *proto.HostInfo
	binary                        core.NginxBinary
	dataplaneSoftwareDetails      map[string]*proto.DataplaneSoftwareDetails
	pipeline                      core.MessagePipeInterface
	dataplaneSoftwareDetailsMutex sync.Mutex
}

func NewOneTimeRegistration(
	config *config.Config,
	binary core.NginxBinary,
	env core.Environment,
	meta *proto.Metadata,
	version string,
) *OneTimeRegistration {
	// this might be slow so do on startup
	host := env.NewHostInfo(version, &config.Tags, config.ConfigDirs, true)
	return &OneTimeRegistration{
		tags:                          &config.Tags,
		agentVersion:                  version,
		meta:                          meta,
		config:                        config,
		env:                           env,
		host:                          host,
		binary:                        binary,
		dataplaneSoftwareDetails:      make(map[string]*proto.DataplaneSoftwareDetails),
		dataplaneSoftwareDetailsMutex: sync.Mutex{},
	}
}

func (r *OneTimeRegistration) Init(pipeline core.MessagePipeInterface) {
	log.Info("OneTimeRegistration initializing")
	r.pipeline = pipeline
	go r.startRegistration()
}

func (r *OneTimeRegistration) Close() {
	log.Info("OneTimeRegistration is wrapping up")
}

func (r *OneTimeRegistration) Info() *core.Info {
	return core.NewInfo("OneTimeRegistration", "v0.0.1")
}

func (r *OneTimeRegistration) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.RegistrationCompletedTopic):
		log.Info("OneTimeRegistration completed")
	case msg.Exact(core.DataplaneSoftwareDetailsUpdated):
		switch data := msg.Data().(type) {
		case *payloads.DataplaneSoftwareDetailsUpdate:
			r.dataplaneSoftwareDetailsMutex.Lock()
			defer r.dataplaneSoftwareDetailsMutex.Unlock()
			r.dataplaneSoftwareDetails[data.GetPluginName()] = data.GetDataplaneSoftwareDetails()
		}
	}
}

func (r *OneTimeRegistration) Subscriptions() []string {
	return []string{
		core.RegistrationCompletedTopic,
		core.DataplaneSoftwareDetailsUpdated,
	}
}

// startRegistration checks in a retry loop if the plugins enabled that transmit dataplane
// software details have transmitted their details to OneTimeRegistration then registers.
// If the plugins do not successfully transmit their details before the max retries is
// reached then an error will be logged then registration will start with whatever
// dataplane software details were successfully transmitted (if any).
func (r *OneTimeRegistration) startRegistration() {
	log.Debug("OneTimeRegistration waiting on dataplane software details to be ready for registration")
	err := sdk.WaitUntil(
		context.Background(), softwareDetailsOperationInterval, softwareDetailsOperationInterval,
		dataplaneSoftwareDetailsMaxWaitTime, r.areDataplaneSoftwareDetailsReady,
	)
	if err != nil {
		log.Warn(err.Error())
	}

	r.registerAgent()
}

// dataplaneSoftwareDetailsReady Determines if all the plugins enabled that transmit dataplane
// software details have transmitted their details to OneTimeRegistration. An error is returned
// if any plugins enabled that transmit dataplane software details have not transmitted their
// details to OneTimeRegistration.
func (r *OneTimeRegistration) areDataplaneSoftwareDetailsReady() error {
	if len(r.config.Extensions) == 0 {
		log.Trace("No extension plugins to register")
		return nil
	}

	r.dataplaneSoftwareDetailsMutex.Lock()
	defer r.dataplaneSoftwareDetailsMutex.Unlock()

	for _, extension := range r.config.Extensions {
		if _, ok := r.dataplaneSoftwareDetails[extension]; !ok {
			return fmt.Errorf("Registration max retries has been met before the extension %s was ready for registration", extension)
		}
	}

	log.Debug("All dataplane software details are ready for registration")
	return nil
}

func (r *OneTimeRegistration) registerAgent() {
	var details []*proto.NginxDetails

	for _, proc := range r.env.Processes() {
		// only need master process for registration
		if proc.IsMaster {
			nginxDetails := r.binary.GetNginxDetailsFromProcess(proc)
			details = append(details, nginxDetails)
			// Reading nginx config during registration to populate nginx fields like access/error logs, etc.
			_, err := r.binary.ReadConfig(nginxDetails.GetConfPath(), nginxDetails.NginxId, r.env.GetSystemUUID())
			if err != nil {
				log.Warnf("Unable to read config for NGINX instance %s, %v", nginxDetails.NginxId, err)
			}
		} else {
			log.Tracef("NGINX non-master process: %d", proc.Pid)
		}
	}
	if len(details) == 0 {
		log.Info("No master process found")
	}
	updated, err := types.TimestampProto(r.config.Updated)
	if err != nil {
		log.Warnf("failed to parse proto timestamp %s: %s, assuming now", r.config.Updated, err)
		updated = types.TimestampNow()
	}
	log.Infof("Registering %s", r.env.GetSystemUUID())

	agentConnectRequest := &proto.Command{
		Meta: r.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_AgentConnectRequest{
			AgentConnectRequest: &proto.AgentConnectRequest{
				Host: r.host,
				Meta: &proto.AgentMeta{
					Version:       r.agentVersion,
					DisplayName:   r.config.DisplayName,
					Tag:           *r.tags,
					InstanceGroup: r.config.InstanceGroup,
					Updated:       updated,
					SystemUid:     r.env.GetSystemUUID(),
					AgentDetails: &proto.AgentDetails{
						Features:   r.config.Features,
						Extensions: r.config.Extensions,
						Tags:       *r.tags,
						Alias:      "",
					},
				},
				Details:                  details,
				DataplaneSoftwareDetails: r.dataplaneSoftwareDetailsSlice(),
			},
		},
	}

	log.Tracef("AgentConnectRequest: %v", agentConnectRequest)

	r.pipeline.Process(
		core.NewMessage(core.CommRegister, agentConnectRequest),
		core.NewMessage(core.RegistrationCompletedTopic, nil),
	)
}

// dataplaneSoftwareDetails converts the map of dataplane software details into a
// slice of dataplane software details and returns it.
func (r *OneTimeRegistration) dataplaneSoftwareDetailsSlice() []*proto.DataplaneSoftwareDetails {
	allDetails := []*proto.DataplaneSoftwareDetails{}

	r.dataplaneSoftwareDetailsMutex.Lock()
	defer r.dataplaneSoftwareDetailsMutex.Unlock()
	for _, details := range r.dataplaneSoftwareDetails {
		if details != nil {
			allDetails = append(allDetails, details)
		}
	}

	return allDetails
}
