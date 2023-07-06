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
	commander  client.Commander
	pipeline   core.MessagePipeInterface
	conf       *config.Config
	env        core.Environment
	binary     core.NginxBinary
	version    string
	featureMap map[string]func(data string)
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

	f.featureMap = map[string]func(data string){
		agent_config.FeatureMetrics: func(data string) {
			f.enableMetricsFeature(data)
		},
		agent_config.FeatureAgentAPI: func(data string) {
			f.enableAgentAPIFeature(data)
		},
		agent_config.FeatureRegistration: func(data string) {
			f.enableRegistrationFeature(data)
		},
		agent_config.FeatureMetricsThrottle: func(data string) {
			f.enableMetricsThrottleFeature(data)
		},
		agent_config.FeatureDataPlaneStatus: func(data string) {
			f.enableDataPlaneStatusFeature(data)
		},
		agent_config.FeatureProcessWatcher: func(data string) {
			f.enableProcessWatcherFeature(data)
		},
		agent_config.FeatureActivityEvents: func(data string) {
			f.enableActivityEventsFeature(data)
		},
		agent_config.FeatureFileWatcher: func(data string) {
			f.enableFileWatcherFeature(data)
		},
		agent_config.FeatureNginxCounting: func(data string) {
			f.enableNginxCountingFeature(data)
		},
		agent_config.FeatureNginxConfigAsync: func(data string) {
			f.enableNginxConfigAsyncFeature(data)
		},
	}
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
	log.Debugf("Process function in the features.go, %s %v", msg.Topic(), msg.Data())

	data := msg.Data()

	if msg.Topic() == core.EnableFeature {
		if initFeature, ok := f.featureMap[data.(string)]; ok {
			initFeature(data.(string))
		}
	}
}

func (f *Features) enableNginxConfigAsyncFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureNginxConfigAsync) {
		conf, err := config.GetConfig(f.conf.ClientID)
		if err != nil {
			log.Warnf("Unable to get agent config, %v", err)
		}
		f.conf = conf

		nginx := NewNginx(f.commander, f.binary, f.env, f.conf)

		err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{nginx}, nil)

		if err != nil {
			log.Warnf("Unable to register %s feature, %v", data, err)
		}

		nginx.Init(f.pipeline)

	}
}

func (f *Features) enableMetricsFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) {
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

func (f *Features) enableAgentAPIFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureAgentAPI) {
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

func (f *Features) enableRegistrationFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureRegistration) {
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

func (f *Features) enableMetricsThrottleFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetricsThrottle) {
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

func (f *Features) enableDataPlaneStatusFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureDataPlaneStatus) {
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

func (f *Features) enableProcessWatcherFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureProcessWatcher) {
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

func (f *Features) enableActivityEventsFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureActivityEvents) {
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

func (f *Features) enableFileWatcherFeature(data string) {
	if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureFileWatcher) {
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
		fileWatcherThrottle.Init(f.pipeline)
	}
}

func (f *Features) enableNginxCountingFeature(data string) {
	if len(f.conf.Nginx.NginxCountingSocket) > 0 {
		if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureNginxCounting) {
			conf, err := config.GetConfig(f.conf.ClientID)
			if err != nil {
				log.Warnf("Unable to get agent config, %v", err)
			}
			f.conf = conf

			if !f.pipeline.IsPluginAlreadyRegistered(agent_config.FeatureMetrics) {
				metrics := NewMetrics(f.conf, f.env, f.binary)
				err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{metrics}, nil)

				if err != nil {
					log.Warnf("Unable to register %s feature, %v", data, err)
				}

				metrics.Init(f.pipeline)
			}
			nginxCounting := NewNginxCounter(f.conf, f.binary, f.env)

			err = f.pipeline.Register(agent_config.DefaultPluginSize, []core.Plugin{nginxCounting}, nil)

			if err != nil {
				log.Warnf("Unable to register %s feature, %v", data, err)
			}

			nginxCounting.Init(f.pipeline)
		}
	}
}
