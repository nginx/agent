/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/nap"
)

const (
	napPluginVersion = "v0.0.1"
	napPluginName    = agent_config.NginxAppProtectExtensionPlugin

	napDegradedMessage = "Nginx App Protect is installed but is not running"

	napDefaultMinInterval = time.Second * 10
)

var nginxAppProtectDefaults = &NginxAppProtectConfig{
	ReportInterval:         1 * time.Minute,
	PrecompiledPublication: false,
}

// NginxAppProtect monitors the NAP installation on the system and reports back its details
type NginxAppProtect struct {
	nap                    nap.NginxAppProtect
	messagePipeline        core.MessagePipeInterface
	env                    core.Environment
	reportInterval         time.Duration
	precompiledPublication bool
	ctx                    context.Context
	ctxCancel              context.CancelFunc
	softwareDetails        *proto.DataplaneSoftwareDetails_AppProtectWafDetails
}

type NginxAppProtectConfig struct {
	ReportInterval         time.Duration `mapstructure:"report_interval" yaml:"-"`
	PrecompiledPublication bool          `mapstructure:"precompiled_publication" yaml:"-"`
}

func NewNginxAppProtect(config *config.Config, env core.Environment, nginxAppProtectConf interface{}) (*NginxAppProtect, error) {
	napTime, err := nap.NewNginxAppProtect(nap.DefaultOptNAPDir, nap.DefaultNMSCompilerDir)
	if err != nil {
		return nil, err
	}

	nginxAppProtectConfig := nginxAppProtectDefaults

	if nginxAppProtectConf != nil {
		var err error
		nginxAppProtectConfig, err = agent_config.DecodeConfig[*NginxAppProtectConfig](nginxAppProtectConf)
		if err != nil {
			log.Errorf("Error decoding configuration for extension plugin %s, %v", napPluginName, err)
			return nil, err
		}
	}

	reportInterval := nginxAppProtectConfig.ReportInterval
	if reportInterval < napDefaultMinInterval {
		reportInterval = napDefaultMinInterval
		log.Warnf(
			"The provided Nginx App Protect report interval (%s) is less than the allowed minimum, updating Nginx App Protect report interval to %s",
			nginxAppProtectConfig.ReportInterval, reportInterval,
		)
	}

	nginxAppProtect := &NginxAppProtect{
		nap:                    *napTime,
		env:                    env,
		reportInterval:         reportInterval,
		precompiledPublication: nginxAppProtectConfig.PrecompiledPublication,
	}

	return nginxAppProtect, nil
}

func (n *NginxAppProtect) Info() *core.Info {
	return core.NewInfo(napPluginName, napPluginVersion)
}

func (n *NginxAppProtect) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", napPluginName)
	n.messagePipeline = pipeline
	ctx, cancel := context.WithCancel(n.messagePipeline.Context())
	n.ctx = ctx
	n.ctxCancel = cancel
	go n.monitor()
}

func (n *NginxAppProtect) Process(msg *core.Message) {}

func (n *NginxAppProtect) Subscriptions() []string {
	return []string{
		core.AgentConfigChanged,
	}
}

func (n *NginxAppProtect) Close() {
	log.Infof("%s is wrapping up", napPluginName)
	n.ctxCancel()
}

// monitor Monitors the system for any changes related to NAP, if any changes are detected
// then a report message is sent through the communication pipeline indicating what the
// previous state of NAP was and what the new state.
func (n *NginxAppProtect) monitor() {
	initialDetails := n.generateNAPDetailsProtoCommand()
	log.Infof("Initial Nginx App Protect details: %+v", initialDetails)
	n.messagePipeline.Process(
		core.NewMessage(
			core.DataplaneSoftwareDetailsUpdated,
			payloads.NewDataplaneSoftwareDetailsUpdate(
				napPluginName,
				&proto.DataplaneSoftwareDetails{
					Data: initialDetails,
				},
			),
		),
	)

	napUpdateChannel := n.nap.Monitor(n.reportInterval)

	for {
		select {
		case updateMsg := <-napUpdateChannel:

			// Communicate the update in NAP status via message pipeline
			log.Infof("Change in NAP detected... Previous: %+v... Updated: %+v", updateMsg.PreviousReport, updateMsg.UpdatedReport)
			napReportMsg := n.generateNAPDetailsProtoCommand()
			n.messagePipeline.Process(
				core.NewMessage(
					core.DataplaneSoftwareDetailsUpdated,
					payloads.NewDataplaneSoftwareDetailsUpdate(
						napPluginName,
						&proto.DataplaneSoftwareDetails{
							Data: napReportMsg,
						},
					),
				),
			)

		case <-time.After(n.reportInterval):
			log.Infof("No NAP changes detected after %v seconds... NAP Values: %+v", n.reportInterval.Seconds(), n.nap.GenerateNAPReport())

		case <-n.ctx.Done():
			return
		}
	}
}

// generateNAPDetailsProtoCommand converts the current NAP report to the proto command
// format for reporting NAP details.
func (n *NginxAppProtect) generateNAPDetailsProtoCommand() *proto.DataplaneSoftwareDetails_AppProtectWafDetails {
	napReport := n.nap.GenerateNAPReport()
	var napStatus proto.AppProtectWAFHealth_AppProtectWAFStatus
	degradedReason := ""

	switch napReport.Status {
	case nap.MISSING.String():
		napStatus = proto.AppProtectWAFHealth_UNKNOWN
	case nap.INSTALLED.String():
		napStatus = proto.AppProtectWAFHealth_DEGRADED
		degradedReason = napDegradedMessage
	case nap.RUNNING.String():
		napStatus = proto.AppProtectWAFHealth_ACTIVE
	}

	n.softwareDetails = &proto.DataplaneSoftwareDetails_AppProtectWafDetails{
		AppProtectWafDetails: &proto.AppProtectWAFDetails{
			WafLocation:             nap.APP_PROTECT_METADATA_FILE_PATH,
			WafVersion:              napReport.NAPVersion,
			AttackSignaturesVersion: napReport.AttackSignaturesVersion,
			ThreatCampaignsVersion:  napReport.ThreatCampaignsVersion,
			Health: &proto.AppProtectWAFHealth{
				SystemId:            n.env.GetSystemUUID(),
				AppProtectWafStatus: napStatus,
				DegradedReason:      degradedReason,
			},
		},
	}

	log.Debugf("Generated NAP details proto message: %+v", n.softwareDetails)

	return n.softwareDetails
}
