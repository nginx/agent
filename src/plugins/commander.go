/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

// Commander plugin is the receiver, dispatcher, and sender of all commands
type Commander struct {
	pipeline core.MessagePipeInterface
	ctx      context.Context
	cmdr     client.Commander
	wg       sync.WaitGroup
	config   *config.Config
}

func NewCommander(cmdr client.Commander, config *config.Config) *Commander {
	return &Commander{
		cmdr:   cmdr,
		wg:     sync.WaitGroup{},
		config: config,
	}
}

func (c *Commander) Init(pipeline core.MessagePipeInterface) {
	c.pipeline = pipeline
	c.ctx = pipeline.Context()
	log.Info("Commander initializing")
	go c.dispatchLoop()
}

func (c *Commander) Close() {
	log.Info("Commander is wrapping up")
}

func (c *Commander) Info() *core.Info {
	return core.NewInfo("Commander", "v0.0.1")
}

func (c *Commander) Subscriptions() []string {
	return []string{core.CommRegister, core.CommStatus, core.CommResponse, core.AgentConnected, core.Events, core.AgentConfig}
}

// Process -
// Agent Communication => Control Plane
// *Command_AgentConnectRequest
// *Command_CmdStatus / CommandStatusResp
// *Command_DataplaneStatus
// *Command_NginxConfigResponse - upload
// *Command_AgentConfigRequest
// *Command_AgentConfig
func (c *Commander) Process(msg *core.Message) {
	log.Tracef("Process function in the commander.go, %s %v", msg.Topic(), msg.Data())
	switch cmd := msg.Data().(type) {
	case *proto.Command:
		switch msg.Topic() {
		case core.CommRegister, core.CommStatus, core.CommResponse, core.Events:
			c.sendCommand(c.ctx, cmd)
		case core.AgentConnected:
			c.agentRegistered(cmd)
		case core.AgentConfig:
			c.agentBackoff(cmd)
		}
	}
}

func (c *Commander) agentBackoff(cmd *proto.Command) {
	log.Debugf("agentBackoff in commander.go with command %v ", cmd)

	if cmd.GetAgentConfig() == nil {
		log.Warnf("update commander client with default backoff settings as agent config nil, for a pause command %+v", cmd)
		c.cmdr.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}
	if cmd.GetAgentConfig().GetDetails() == nil {
		log.Warnf("update commander client with default backoff settings as agent details nil, for a pause command %+v", cmd)
		c.cmdr.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}
	if cmd.GetAgentConfig().GetDetails().GetServer() == nil {
		log.Warnf("update commander client with default backoff settings as server nil, for a pause command %+v", cmd)
		c.cmdr.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}

	backoffSetting := cmd.GetAgentConfig().GetDetails().GetServer().Backoff
	if backoffSetting == nil {
		log.Warnf("update commander client with default backoff settings as backoff settings nil, for a pause command %+v", cmd)
		c.cmdr.WithBackoffSettings(client.DefaultBackoffSettings)
		return
	}

	multiplier := backoff.BACKOFF_MULTIPLIER
	if backoffSetting.GetMultiplier() != 0 {
		multiplier = backoffSetting.GetMultiplier()
	}

	jitter := backoff.BACKOFF_JITTER
	if backoffSetting.GetRandomizationFactor() != 0 {
		jitter = backoffSetting.GetRandomizationFactor()
	}

	cBackoff := backoff.BackoffSettings{
		InitialInterval: time.Duration(backoffSetting.InitialInterval * int64(time.Second)),
		MaxInterval:     time.Duration(backoffSetting.MaxInterval * int64(time.Second)),
		MaxElapsedTime:  time.Duration(backoffSetting.MaxElapsedTime * int64(time.Second)),
		Multiplier:      multiplier,
		Jitter:          jitter,
	}
	log.Debugf("update commander client backoff settings to %+v, for a pause command %+v", cBackoff, cmd)
	c.cmdr.WithBackoffSettings(cBackoff)
}

func (c *Commander) agentRegistered(cmd *proto.Command) {
	switch commandData := cmd.Data.(type) {
	case *proto.Command_AgentConnectResponse:
		log.Infof("config command %v", commandData)
		if agtCfg := commandData.AgentConnectResponse.AgentConfig; agtCfg != nil &&
			agtCfg.Configs != nil && len(agtCfg.Configs.Configs) > 0 {
			// Update config tags and features if they were out of sync between Manager and Agent
			if agtCfg.Details != nil && (len(agtCfg.Details.Tags) > 0 || len(agtCfg.Details.Features) > 0) {
				configUpdated, err := config.UpdateAgentConfig(c.config.ClientID, agtCfg.Details.Tags, agtCfg.Details.Features)
				if err != nil {
					log.Errorf("Failed updating Agent config - %v", err)
				}

				// If the config was updated send a new agent config updated message
				if configUpdated {
					c.pipeline.Process(core.NewMessage(core.AgentConfigChanged, ""))
				}
			}

			for _, config := range agtCfg.Configs.Configs {
				c.pipeline.Process(core.NewMessage(core.NginxConfigUpload, config))
			}

			if agtCfg.Details != nil && agtCfg.Details.Extensions != nil {
				for _, extension := range agtCfg.Details.Extensions {
					if extension == agent_config.AdvancedMetricsExtensionPlugin ||
						extension == config.NginxAppProtectKey ||
						extension == config.NAPMonitoringKey {
						c.pipeline.Process(core.NewMessage(core.EnableExtension, extension))
					}
				}
			}

			if agtCfg.Details != nil && agtCfg.Details.Features != nil {
				for index, feature := range agtCfg.Details.Features {
					agtCfg.Details.Features[index] = strings.Replace(feature, "features_", "", 1)
				}

				sort.Strings(agtCfg.Details.Features)
			}

			sort.Strings(c.config.Features)

			synchronizedFeatures := reflect.DeepEqual(agtCfg.Details.Features, c.config.Features)

			if !synchronizedFeatures {
				for _, feature := range c.config.Features {
					if feature != agent_config.FeatureRegistration {
						c.deRegisterPlugin(feature)
					}

				}
			}

			if agtCfg.Details != nil && agtCfg.Details.Features != nil && !synchronizedFeatures {
				for _, feature := range agtCfg.Details.Features {
					c.pipeline.Process(core.NewMessage(core.EnableFeature, feature))
				}
			}
		}

	default:
		log.Debugf("unhandled command: %T", cmd.Data)
	}
}

func (c *Commander) deRegisterPlugin(data string) {
	if data == agent_config.FeatureFileWatcher {

		err := c.pipeline.DeRegister([]string{agent_config.FeatureFileWatcher, agent_config.FeatureFileWatcherThrottle})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}

	} else if data == agent_config.FeatureNginxConfigAsync {

		err := c.pipeline.DeRegister([]string{"NginxBinary"})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}

	} else {
		err := c.pipeline.DeRegister([]string{data})
		if err != nil {
			log.Warnf("Error Deregistering %v Plugin: %v", data, err)
		}
	}
}

func (c *Commander) sendCommand(ctx context.Context, cmd *proto.Command) {
	log.Debugf("Sending command (messageId=%s), %v", cmd.GetMeta().MessageId, cmd.GetData())
	if err := c.cmdr.Send(ctx, client.MessageFromCommand(cmd)); err != nil {
		log.Errorf("Error sending to command channel %v", err)
	}
}

func (c *Commander) dispatchLoop() {
	c.wg.Add(1)
	defer c.wg.Done()
	var ok bool
	for {
		var cmd *proto.Command
		select {
		case <-c.ctx.Done():
			log.Debug("cmdr dispatch loop exiting")
			err := c.ctx.Err()
			if err != nil {
				log.Errorf("error in done context commander dispatchLoop %v", err)
			}
			return
		case msg := <-c.cmdr.Recv():
			switch msg.Classification() {
			case client.MsgClassificationCommand:
				if cmd, ok = msg.Raw().(*proto.Command); !ok {
					log.Warnf("expected Command type, but got: %T", msg.Raw())
					continue
				}
			default:
				log.Warnf("expected %T type, but got: %T", &proto.Command{}, msg.Raw())
				continue
			}
		}

		log.Debugf("Command msg from data plane: %v", cmd)
		var topic string
		switch cmd.Data.(type) {
		case *proto.Command_NginxConfig, *proto.Command_NginxConfigResponse:
			topic = core.CommNginxConfig
		case *proto.Command_AgentConnectRequest, *proto.Command_AgentConnectResponse:
			topic = core.AgentConnected
		case *proto.Command_AgentConfigRequest, *proto.Command_AgentConfig:
			log.Debugf("agent config %T command data type received and ignored", cmd.Data)
			topic = core.AgentConfig
		case *proto.Command_CmdStatus:
			data := cmd.Data.(*proto.Command_CmdStatus)
			if data.CmdStatus.Status != proto.CommandStatusResponse_CMD_OK {
				log.Debugf("command status %T :: %+v", cmd.Data, cmd.Data)
			}
			topic = core.UNKNOWN
			continue
		default:
			if cmd.Data != nil {
				log.Infof("unknown %T command data type received", cmd.Data)
			}
			topic = core.UNKNOWN
			continue
		}

		c.pipeline.Process(core.NewMessage(topic, cmd))
	}
}
