/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"github.com/google/uuid"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

type Features struct {
	pipeline core.MessagePipeInterface
	conf     *config.Config
	env      core.Environment
	binary   core.NginxBinary
	version  string
}

func NewFeatures(conf *config.Config, env core.Environment, binary core.NginxBinary, version string) *Features {
	return &Features{
		conf:    conf,
		env:     env,
		binary:  binary,
		version: version,
	}
}

func (f *Features) Init(pipeline core.MessagePipeInterface) {
	log.Info("Features initializing")
	f.pipeline = pipeline
}

func (f *Features) Close() {
	log.Info("Features is wrapping up")
}

func (f *Features) Info() *core.Info {
	return core.NewInfo("Features Plugin", "v0.0.1")
}

func (f *Features) Subscriptions() []string {
	return []string{
		core.EnableFeature,
	}
}

func (f *Features) Process(msg *core.Message) {
	log.Debugf("--------> Process function in the features.go, %s %v", msg.Topic(), msg.Data())
	switch data := msg.Data().(type) {
	case string:
		switch msg.Topic() {
		case core.EnableFeature:
			log.Info("---------> Data")
			log.Info(data)
			if data == agent_config.FeatureMetrics {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureMetrics) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					metrics := NewMetrics(f.conf, f.env, f.binary)

					log.Info()
					log.Info("-------> Metrics Feature is not enabled registering ")

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{metrics}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					metrics.Init(f.pipeline)

				}
			}
			if data == agent_config.FeatureAgentAPI {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureAgentAPI) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					api := NewAgentAPI(f.conf, f.env, f.binary)

					log.Info()
					log.Info(" --------> Agent Api Feature is not enabled registering ")

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{api}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					api.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureRegistration {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureAgentAPI) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					registration := NewOneTimeRegistration(f.conf, f.binary, f.env, sdkGRPC.NewMessageMeta(uuid.NewString()), f.version)

					log.Info()
					log.Info(" --------> Registration Feature is not enabled registering ")

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{registration}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					registration.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureMetricsThrottle {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureMetricsThrottle) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					metricsThrottle := NewMetricsThrottle(f.conf, f.env)

					log.Info()
					log.Info(" --------> Metrics Throttle Feature is not enabled registering ")

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{metricsThrottle}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					metricsThrottle.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureDataPlaneStatus {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureMetricsThrottle) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					dataPlaneStatus := NewDataPlaneStatus(f.conf, sdkGRPC.NewMessageMeta(uuid.NewString()), f.binary, f.env, f.version)

					log.Info()
					log.Info(" --------> Metrics Throttle Feature is not enabled registering ")

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{dataPlaneStatus}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					dataPlaneStatus.Init(f.pipeline)

				}
			}
		}
	}
}

func (f *Features) isPluginAlreadyRegistered(pluginName string) bool {
	pluginAlreadyRegistered := false
	for _, plugin := range f.pipeline.GetPlugins() {
		if plugin.Info().Name() == pluginName {
			pluginAlreadyRegistered = true
		}
	}
	return pluginAlreadyRegistered
}
