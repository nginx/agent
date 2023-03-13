/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

// ConfigReader reads in configuration from the messagePipe
type ConfigReader struct {
	messagePipeline core.MessagePipeInterface
	config          *config.Config
}

func NewConfigReader(config *config.Config) *ConfigReader {
	conf := &ConfigReader{config: config}

	return conf
}

func (r *ConfigReader) Init(pipeline core.MessagePipeInterface) {
	r.messagePipeline = pipeline
}

func (r *ConfigReader) Info() *core.Info {
	return core.NewInfo("Config Reader", "v0.0.1")
}

func (r *ConfigReader) Close() {
	log.Info("ConfigReader is wrapping up")
}

func (r *ConfigReader) Process(msg *core.Message) {
	// if this is a map or any other config we might have to handle that case
	if msg.Match("configs.agent.") {
		str, ok := msg.Data().(string)
		if ok {
			data := viper.GetString(str)
			if data != "" {
				r.messagePipeline.Process(core.NewMessage(str, data))
			} else {
				r.messagePipeline.Process(core.NewMessage(str, ""))
				log.Errorf("Data not in the config for key %s: %v", str, data)
			}
		} else {
			log.Infof("Unable to cast %v to string", msg.Data())
		}
		return
	}

	switch cmd := msg.Data().(type) {
	case *proto.Command:
		switch msg.Topic() {
		case core.AgentConfig:
			// Update the agent config on disk
			r.updateAgentConfig(cmd)
		}
	}
}

func (r *ConfigReader) Subscriptions() []string {
	return []string{core.CommMetrics, core.AgentConfig, core.AgentConfigChanged}
}

func (r *ConfigReader) updateAgentConfig(cmd *proto.Command) {
	switch commandData := cmd.Data.(type) {
	case *proto.Command_AgentConfig:
		configUpdated, err := config.UpdateAgentConfig(r.config.ClientID, commandData.AgentConfig.Details.Tags, commandData.AgentConfig.Details.Features)
		if err != nil {
			log.Errorf("Failed updating Agent config - %v", err)
		}

		// If the config was updated send a new agent config updated message
		if configUpdated {
			log.Debugf("Updated agent config on disk")
			r.messagePipeline.Process(core.NewMessage(core.AgentConfigChanged, ""))
		}

		if commandData.AgentConfig.Details != nil && commandData.AgentConfig.Details.Extensions != nil {
			for _, extension := range commandData.AgentConfig.Details.Extensions {
				if extension == agent_config.AdvancedMetricsExtensionPlugin ||
					extension == agent_config.NginxAppProtectExtensionPlugin ||
					extension == agent_config.NginxAppProtectMonitoringExtensionPlugin {
					r.messagePipeline.Process(core.NewMessage(core.EnableExtension, extension))
				}
			}
		}
	}
}
