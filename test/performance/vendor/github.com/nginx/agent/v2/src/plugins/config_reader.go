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
	"reflect"
	"sort"
	"strings"
	"sync"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

// ConfigReader reads in configuration from the messagePipe
type ConfigReader struct {
	messagePipeline core.MessagePipeInterface
	config          *config.Config
	mu              sync.RWMutex
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
		case core.AgentConfig, core.AgentConnected:
			// Update the agent config on disk
			switch commandData := cmd.Data.(type) {
			case *proto.Command_AgentConfig:
				r.updateAgentConfig(commandData.AgentConfig)
			case *proto.Command_AgentConnectResponse:
				r.updateAgentConfig(commandData.AgentConnectResponse.AgentConfig)
			}
		}
	}
}

func (r *ConfigReader) Subscriptions() []string {
	return []string{core.CommMetrics, core.AgentConfig, core.AgentConfigChanged, core.AgentConnected}
}

func (r *ConfigReader) updateAgentConfig(payloadAgentConfig *proto.AgentConfig) {
	if payloadAgentConfig != nil && payloadAgentConfig.Details != nil {

		onDiskAgentConfig, _ := config.GetConfig(r.config.ClientID)
		synchronizeFeatures := false
		synchronizeTags := false

		if payloadAgentConfig.Details.Features != nil {

			for index, feature := range payloadAgentConfig.Details.Features {
				payloadAgentConfig.Details.Features[index] = strings.Replace(feature, "features_", "", 1)
			}

			sort.Strings(onDiskAgentConfig.Features)
			sort.Strings(payloadAgentConfig.Details.Features)
			synchronizeFeatures = !reflect.DeepEqual(payloadAgentConfig.Details.Features, onDiskAgentConfig.Features)

		}
		
		if payloadAgentConfig.Details.Tags != nil {
			sort.Strings(onDiskAgentConfig.Tags)
			sort.Strings(payloadAgentConfig.Details.Tags)
			synchronizeTags = !reflect.DeepEqual(payloadAgentConfig.Details.Tags, onDiskAgentConfig.Tags)
		}

		if (synchronizeFeatures || synchronizeTags) {
			configUpdated, err := config.UpdateAgentConfig(r.config.ClientID, payloadAgentConfig.Details.Tags, payloadAgentConfig.Details.Features)
			if err != nil {
				log.Errorf("Failed updating Agent config - %v", err)
			}

			// If the config was updated send a new agent config updated message
			if configUpdated {
				log.Debugf("Updated agent config on disk")
				r.messagePipeline.Process(core.NewMessage(core.AgentConfigChanged, ""))
			}
		}

		if synchronizeFeatures {
			r.synchronizeFeatures(payloadAgentConfig)
		}

		if payloadAgentConfig.Details.Extensions != nil {
			for _, extension := range payloadAgentConfig.Details.Extensions {
				if extension == agent_config.AdvancedMetricsExtensionPlugin ||
					extension == agent_config.NginxAppProtectExtensionPlugin ||
					extension == agent_config.NginxAppProtectMonitoringExtensionPlugin {
					r.messagePipeline.Process(core.NewMessage(core.EnableExtension, extension))
				}
			}
		}
	}
}

func (r *ConfigReader) synchronizeFeatures(agtCfg *proto.AgentConfig) {
	if r.config != nil {
		for _, feature := range r.config.Features {
			if feature != agent_config.FeatureRegistration {
				r.deRegisterPlugin(feature)
			}
		}
	}

	if agtCfg.Details != nil {
		for _, feature := range agtCfg.Details.Features {
			r.messagePipeline.Process(core.NewMessage(core.EnableFeature, feature))
		}
	}
}

func (r *ConfigReader) deRegisterPlugin(data string) {
	if data == agent_config.FeatureFileWatcher {

		err := r.messagePipeline.DeRegister([]string{agent_config.FeatureFileWatcher, agent_config.FeatureFileWatcherThrottle})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}

	} else if data == agent_config.FeatureNginxConfigAsync {

		err := r.messagePipeline.DeRegister([]string{"NginxBinary"})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}

	} else {
		err := r.messagePipeline.DeRegister([]string{data})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}
	}
}
