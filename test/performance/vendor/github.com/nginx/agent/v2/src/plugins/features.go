/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Features struct {
	commander       client.Commander
	pipeline        core.MessagePipeInterface
	conf            *config.Config
	env             core.Environment
	binary          core.NginxBinary
	version         string
	featureMap      map[string]func(data string) []core.Plugin
	processes       []*core.Process
	agentEventsMeta *events.AgentEventMeta
}

func NewFeatures(
	commander client.Commander,
	conf *config.Config,
	env core.Environment,
	binary core.NginxBinary,
	version string,
	processes []*core.Process,
	agentEventsMeta *events.AgentEventMeta,
) *Features {
	return &Features{
		commander:       commander,
		conf:            conf,
		env:             env,
		binary:          binary,
		version:         version,
		processes:       processes,
		agentEventsMeta: agentEventsMeta,
	}
}

func (f *Features) Init(pipeline core.MessagePipeInterface) {
	log.Info("Features initializing")
	f.pipeline = pipeline

	f.featureMap = map[string]func(data string) []core.Plugin{
		agent_config.FeatureMetrics: func(data string) []core.Plugin {
			return f.enableMetricsFeature(data)
		},
		agent_config.FeatureAgentAPI: func(data string) []core.Plugin {
			return f.enableAgentAPIFeature(data)
		},
		agent_config.FeatureRegistration: func(data string) []core.Plugin {
			return f.enableRegistrationFeature(data)
		},
		agent_config.FeatureMetricsThrottle: func(data string) []core.Plugin {
			return f.enableMetricsThrottleFeature(data)
		},
		agent_config.FeatureMetricsSender: func(data string) []core.Plugin {
			return f.enableMetricsSenderFeature(data)
		},
		agent_config.FeatureMetricsCollection: func(data string) []core.Plugin {
			return f.enableMetricsCollectionFeature(data)
		},
		agent_config.FeatureDataPlaneStatus: func(data string) []core.Plugin {
			return f.enableDataPlaneStatusFeature(data)
		},
		agent_config.FeatureProcessWatcher: func(data string) []core.Plugin {
			return f.enableProcessWatcherFeature(data)
		},
		agent_config.FeatureActivityEvents: func(data string) []core.Plugin {
			return f.enableActivityEventsFeature(data)
		},
		agent_config.FeatureFileWatcher: func(data string) []core.Plugin {
			return f.enableFileWatcherFeature(data)
		},
		agent_config.FeatureNginxCounting: func(data string) []core.Plugin {
			return f.enableNginxCountingFeature(data)
		},
	}
}

func (f *Features) Close() {
	log.Info("Features is wrapping up")
}

func (f *Features) Info() *core.Info {
	return core.NewInfo(agent_config.FeaturesPlugin, "v0.0.1")
}

func (f *Features) Subscriptions() []string {
	return []string{
		core.EnableFeature,
	}
}

func (f *Features) Process(msg *core.Message) {
	log.Infof("Process function in the features.go, %s %v", msg.Topic(), msg.Data())

	data := msg.Data()
	plugins := []core.Plugin{}

	if msg.Topic() == core.EnableFeature {
		for _, feature := range data.([]string) {
			if initFeature, ok := f.featureMap[feature]; ok {
				featurePlugins := initFeature(feature)
				plugins = append(plugins, featurePlugins...)
			}
		}

		err := f.pipeline.Register(f.conf.QueueSize, plugins, nil)
		if err != nil {
			log.Warnf("Unable to register features: %v", err)
		}

		for _, plugin := range plugins {
			plugin.Init(f.pipeline)
		}
	} else if msg.Topic() == core.NginxDetailProcUpdate {
		f.processes = msg.Data().([]*core.Process)
	}
}

func (f *Features) enableMetricsFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) {
		log.Debugf("features.go: enabling metrics feature")
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		metrics := NewMetrics(f.conf, f.env, f.binary, f.processes)
		metricsThrottle := NewMetricsThrottle(f.conf, f.env)
		metricsSender := NewMetricsSender(f.commander, conf)

		return []core.Plugin{metrics, metricsThrottle, metricsSender}
	}
	return []core.Plugin{}
}

func (f *Features) enableMetricsCollectionFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) &&
		!f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetricsCollection) {

		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		metrics := NewMetrics(f.conf, f.env, f.binary, f.processes)

		return []core.Plugin{metrics}
	}
	return []core.Plugin{}
}

func (f *Features) enableMetricsThrottleFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) &&
		!f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetricsThrottle) {

		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		metricsThrottle := NewMetricsThrottle(f.conf, f.env)

		return []core.Plugin{metricsThrottle}
	}
	return []core.Plugin{}
}

func (f *Features) enableMetricsSenderFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) &&
		!f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetricsSender) {

		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		metricsSender := NewMetricsSender(f.commander, conf)

		return []core.Plugin{metricsSender}
	}
	return []core.Plugin{}
}

func (f *Features) enableAgentAPIFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureAgentAPI) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		api := NewAgentAPI(f.conf, f.env, f.binary, f.processes)

		return []core.Plugin{api}
	}
	return []core.Plugin{}
}

func (f *Features) enableRegistrationFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureRegistration) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		registration := NewOneTimeRegistration(f.conf, f.binary, f.env, sdkGRPC.NewMessageMeta(uuid.NewString()), f.processes)

		return []core.Plugin{registration}
	}
	return []core.Plugin{}
}

func (f *Features) enableDataPlaneStatusFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureDataPlaneStatus) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		dataPlaneStatus := NewDataPlaneStatus(f.conf, sdkGRPC.NewMessageMeta(uuid.NewString()), f.binary, f.env, f.processes)

		return []core.Plugin{dataPlaneStatus}
	}
	return []core.Plugin{}
}

func (f *Features) enableProcessWatcherFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureProcessWatcher) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		processWatcher := NewProcessWatcher(f.env, f.binary, f.processes, f.conf)

		return []core.Plugin{processWatcher}
	}
	return []core.Plugin{}
}

func (f *Features) enableActivityEventsFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureActivityEvents) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		events := NewEvents(f.conf, f.env, sdkGRPC.NewMessageMeta(uuid.NewString()), f.binary, f.agentEventsMeta)

		return []core.Plugin{events}
	}
	return []core.Plugin{}
}

func (f *Features) enableFileWatcherFeature(_ string) []core.Plugin {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureFileWatcher) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		fileWatcher := NewFileWatcher(f.conf, f.env)
		fileWatcherThrottle := NewFileWatchThrottle()

		return []core.Plugin{fileWatcher, fileWatcherThrottle}
	}
	return []core.Plugin{}
}

func (f *Features) enableNginxCountingFeature(_ string) []core.Plugin {
	countingPlugins := []core.Plugin{}
	if len(f.conf.Nginx.NginxCountingSocket) > 0 {
		if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureNginxCounting) {
			conf, err := config.GetConfig(f.conf.ClientID)
			if err != nil {
				log.Warnf("Unable to get agent config, %v", err)
			}
			f.conf = conf

			if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) &&
				!f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetricsCollection) {

				metrics := NewMetrics(f.conf, f.env, f.binary, f.processes)

				countingPlugins = append(countingPlugins, metrics)
			}

			nginxCounting := NewNginxCounter(f.conf, f.binary, f.env)
			countingPlugins = append(countingPlugins, nginxCounting)

			return countingPlugins
		}
		return []core.Plugin{}
	}
	return []core.Plugin{}
}
