package plugins

import (
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
)

const (
	DEFAULT_PLUGIN_SIZE = 100
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
			if data == config.AdvancedMetricsKey {
				if !e.isPluginAlreadyRegistered(advancedMetricsPluginName) {
					config.SetAdvancedMetricsDefaults()
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					advancedMetrics := NewAdvancedMetrics(e.env, e.conf)
					err = e.pipeline.Register(DEFAULT_PLUGIN_SIZE, advancedMetrics)
					if err != nil {
						log.Warnf("Unable to register %s extension, %v", data, err)
					}
					advancedMetrics.Init(e.pipeline)
				}
			} else if data == config.NginxAppProtectKey {
				if !e.isPluginAlreadyRegistered(napPluginName) {
					config.SetNginxAppProtectDefaults()
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					nap, err := NewNginxAppProtect(e.conf, e.env)
					if err != nil {
						log.Warnf("Unable to load the Nginx App Protect plugin due to the following error: %v", err)
					}
					err = e.pipeline.Register(DEFAULT_PLUGIN_SIZE, nap)
					if err != nil {
						log.Errorf("Unable to register %s extension, %v", data, err)
					}
					nap.Init(e.pipeline)
				}
			} else if data == config.NAPMonitoringKey {
				if !e.isPluginAlreadyRegistered(napMonitoringPluginName) {
					config.SetNAPMonitoringDefaults()
					conf, err := config.GetConfig(e.conf.ClientID)
					if err != nil {
						log.Warnf("Unable to get agent config, %v", err)
					}
					e.conf = conf

					napMonitoring, err := NewNapMonitoring(e.conf)
					if err != nil {
						log.Warnf("Unable to load the Nginx App Protect Monitoring plugin due to the following error: %v", err)
						break
					}
					err = e.pipeline.Register(DEFAULT_PLUGIN_SIZE, napMonitoring)
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
	for _, plugin := range e.pipeline.GetPlugins() {
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
