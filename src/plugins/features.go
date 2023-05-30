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
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

type Features struct {
	commander client.Commander
	pipeline  core.MessagePipeInterface
	conf      *config.Config
	env       core.Environment
	binary    core.NginxBinary
	version   string
}

func NewFeatures(commander client.Commander, conf *config.Config, env core.Environment, binary core.NginxBinary, version string) *Features {
	return &Features{
		commander: commander,
		conf:      conf,
		env:       env,
		binary:    binary,
		version:   version,
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
		core.DisableFeature,
	}
}

func (f *Features) Process(msg *core.Message) {
	log.Debugf("Process function in the features.go, %s %v", msg.Topic(), msg.Data())
	switch data := msg.Data().(type) {
	case string:
		switch msg.Topic() {
		case core.EnableFeature:
			if data == agent_config.FeatureMetrics {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureMetrics) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					metrics := NewMetrics(f.conf, f.env, f.binary)

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

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{metricsThrottle}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					metricsThrottle.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureDataPlaneStatus {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureDataPlaneStatus) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					dataPlaneStatus := NewDataPlaneStatus(f.conf, sdkGRPC.NewMessageMeta(uuid.NewString()), f.binary, f.env, f.version)

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{dataPlaneStatus}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					dataPlaneStatus.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureProcessWatcher {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureProcessWatcher) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					processWatcher := NewProcessWatcher(f.env, f.binary)

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{processWatcher}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					processWatcher.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureActivityEvents {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureActivityEvents) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					events := NewEvents(f.conf, f.env, sdkGRPC.NewMessageMeta(uuid.NewString()), f.binary)

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{events}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					events.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureFileWatcher {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureFileWatcher) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					fileWatcher := NewFileWatcher(f.conf, f.env)
					fileWatcherThrottle := NewFileWatchThrottle()

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{fileWatcher, fileWatcherThrottle}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					fileWatcher.Init(f.pipeline)

				}
			}

			if data == agent_config.FeatureNginxCounting && len(f.conf.Nginx.NginxCountingSocket) > 0 {
				if !f.isPluginAlreadyRegistered(agent_config.FeatureNginxCounting) {
					conf, err := config.GetConfig(f.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					f.conf = conf

					nginxCounting := NewNginxCounter(f.conf, f.binary, f.env)
					metrics := NewMetrics(f.conf, f.env, f.binary)

					err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{metrics, nginxCounting}, nil)

					if err != nil {
						log.Warnf("Unable to register %s feature, %v", data, err)
					}

					nginxCounting.Init(f.pipeline)

				}
			}
		case core.DisableFeature:
			if data == agent_config.FeatureMetrics {
				err := f.pipeline.Deregister([]string{agent_config.FeatureMetrics})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureAgentAPI {
				err := f.pipeline.Deregister([]string{agent_config.FeatureAgentAPI})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}

			if data == agent_config.FeatureRegistration {
				err := f.pipeline.Deregister([]string{agent_config.FeatureRegistration})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureMetricsThrottle {
				err := f.pipeline.Deregister([]string{agent_config.FeatureMetricsThrottle})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureDataPlaneStatus {
				err := f.pipeline.Deregister([]string{agent_config.FeatureDataPlaneStatus})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureProcessWatcher {
				err := f.pipeline.Deregister([]string{agent_config.FeatureProcessWatcher})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}

			}
			if data == agent_config.FeatureActivityEvents {
				err := f.pipeline.Deregister([]string{agent_config.FeatureActivityEvents})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureFileWatcher {
				err := f.pipeline.Deregister([]string{agent_config.FeatureFileWatcher, agent_config.FeatureFileWatcherThrottle})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
				}
			}
			if data == agent_config.FeatureNginxCounting {
				err := f.pipeline.Deregister([]string{agent_config.FeatureNginxCounting})
				if err != nil {
					log.Warnf("Error Deregistering %v Plugin: %v", data, err)
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
