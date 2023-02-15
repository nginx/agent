/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
)

type Extensions struct {
	pipeline core.MessagePipeInterface
	conf     *config.Config
	env      core.Environment
}

func NewExtensions(conf *config.Config, env core.Environment) *Extensions {
	return &Extensions{
		conf: conf,
		env:  env,
	}
}

func (e *Extensions) Init(pipeline core.MessagePipeInterface) {
	log.Info("Extensions initializing")
	e.pipeline = pipeline
}

func (e *Extensions) Close() {
	log.Info("Extensions is wrapping up")
}

func (e *Extensions) Process(msg *core.Message) {
	log.Debugf("Process function in the extensions.go, %s %v", msg.Topic(), msg.Data())
	switch data := msg.Data().(type) {
	case string:
		switch msg.Topic() {
		case core.EnableExtension:
			if data == agent_config.AdvancedMetricsExtensionPlugin {
				if !e.isPluginAlreadyRegistered(agent_config.AdvancedMetricsExtensionPlugin) {
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					advancedMetrics := extensions.NewAdvancedMetrics(
						e.env,
						e.conf,
						config.Viper.Get(agent_config.AdvancedMetricsExtensionPluginConfigKey),
					)
					err = e.pipeline.Register(agent_config.DefaultPluginSize, nil, []core.ExtensionPlugin{advancedMetrics})
					if err != nil {
						log.Warnf("Unable to register %s extension, %v", data, err)
					}
					advancedMetrics.Init(e.pipeline)
				}
			} else if data == agent_config.NginxAppProtectExtensionPlugin {
				if !e.isPluginAlreadyRegistered(agent_config.NginxAppProtectExtensionPlugin) {
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					nap, err := extensions.NewNginxAppProtect(e.conf, e.env, config.Viper.Get(agent_config.NginxAppProtectExtensionPluginConfigKey))
					if err != nil {
						log.Warnf("Unable to load the Nginx App Protect plugin due to the following error: %v", err)
					}
					err = e.pipeline.Register(agent_config.DefaultPluginSize, nil, []core.ExtensionPlugin{nap})
					if err != nil {
						log.Errorf("Unable to register %s extension, %v", data, err)
					}
					nap.Init(e.pipeline)
				}
			} else if data == agent_config.NginxAppProtectMonitoringExtensionPlugin {
				if !e.isPluginAlreadyRegistered(agent_config.NginxAppProtectMonitoringExtensionPlugin) {
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					napMonitoring, err := extensions.NewNAPMonitoring(e.env, e.conf, config.Viper.Get(agent_config.NginxAppProtectMonitoringExtensionPluginConfigKey))
					if err != nil {
						log.Warnf("Unable to load the Nginx App Protect Monitoring plugin due to the following error: %v", err)
						break
					}
					err = e.pipeline.Register(agent_config.DefaultPluginSize, nil, []core.ExtensionPlugin{napMonitoring})
					if err != nil {
						log.Errorf("Unable to register %s extension, %v", data, err)
					}
					napMonitoring.Init(e.pipeline)
				}
			}
		}
	}
}

func (e *Extensions) isPluginAlreadyRegistered(pluginName string) bool {
	pluginAlreadyRegistered := false
	for _, plugin := range e.pipeline.GetExtensionPlugins() {
		if plugin.Info().Name() == pluginName {
			pluginAlreadyRegistered = true
		}
	}
	return pluginAlreadyRegistered
}

func (e *Extensions) Info() *core.Info {
	return core.NewInfo("Extensions Plugin", "v0.0.1")
}

func (e *Extensions) Subscriptions() []string {
	return []string{
		core.EnableExtension,
	}
}
