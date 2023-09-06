package plugins

import (
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions"
	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"

	"github.com/google/uuid"
)

func LoadPlugins(commander client.Commander, binary core.NginxBinary, env core.Environment, reporter client.MetricReporter, loadedConfig *config.Config) ([]core.Plugin, []core.ExtensionPlugin) {
	var corePlugins []core.Plugin
	var extensionPlugins []core.ExtensionPlugin

	if commander != nil {
		corePlugins = append(corePlugins,
			NewCommander(commander, loadedConfig),
		)

		if loadedConfig.IsFeatureEnabled(agent_config.FeatureFileWatcher) {
			corePlugins = append(corePlugins,
				NewFileWatcher(loadedConfig, env),
				NewFileWatchThrottle(),
			)
		}
	}

	if (loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsSender)) && reporter != nil {
		corePlugins = append(corePlugins,
			NewMetricsSender(reporter),
		)
	}

	corePlugins = append(corePlugins,
		NewConfigReader(loadedConfig),
		NewNginx(commander, binary, env, loadedConfig),
		NewExtensions(loadedConfig, env),
		NewFeatures(commander, loadedConfig, env, binary, loadedConfig.Version),
	)

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureRegistration) {
		corePlugins = append(corePlugins, NewOneTimeRegistration(loadedConfig, binary, env, sdkGRPC.NewMessageMeta(uuid.NewString())))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsCollection) ||
		(len(loadedConfig.Nginx.NginxCountingSocket) > 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureNginxCounting)) {
		corePlugins = append(corePlugins, NewMetrics(loadedConfig, env, binary))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsThrottle) {
		corePlugins = append(corePlugins, NewMetricsThrottle(loadedConfig, env))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureDataPlaneStatus) {
		corePlugins = append(corePlugins, NewDataPlaneStatus(loadedConfig, sdkGRPC.NewMessageMeta(uuid.NewString()), binary, env))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureProcessWatcher) {
		corePlugins = append(corePlugins, NewProcessWatcher(env, binary))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureActivityEvents) {
		corePlugins = append(corePlugins, NewEvents(loadedConfig, env, sdkGRPC.NewMessageMeta(uuid.NewString()), binary))
	}

	if loadedConfig.AgentAPI.Port != 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureAgentAPI) {
		corePlugins = append(corePlugins, NewAgentAPI(loadedConfig, env, binary))
	} else {
		log.Info("Agent API not configured")
	}

	if len(loadedConfig.Nginx.NginxCountingSocket) > 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureNginxCounting) {
		corePlugins = append(corePlugins, NewNginxCounter(loadedConfig, binary, env))
	}

	if loadedConfig.Extensions != nil && len(loadedConfig.Extensions) > 0 {
		for _, extension := range loadedConfig.Extensions {
			switch {
			case extension == agent_config.AdvancedMetricsExtensionPlugin:
				advancedMetricsExtensionPlugin := extensions.NewAdvancedMetrics(env, loadedConfig, config.Viper.Get(agent_config.AdvancedMetricsExtensionPluginConfigKey))
				extensionPlugins = append(extensionPlugins, advancedMetricsExtensionPlugin)
			case extension == agent_config.NginxAppProtectExtensionPlugin:
				nginxAppProtectExtensionPlugin, err := extensions.NewNginxAppProtect(loadedConfig, env, config.Viper.Get(agent_config.NginxAppProtectExtensionPluginConfigKey))
				if err != nil {
					log.Errorf("Unable to load the Nginx App Protect plugin due to the following error: %v", err)
				} else {
					extensionPlugins = append(extensionPlugins, nginxAppProtectExtensionPlugin)
				}
			case extension == agent_config.NginxAppProtectMonitoringExtensionPlugin:
				nginxAppProtectMonitoringExtensionPlugin, err := extensions.NewNAPMonitoring(env, loadedConfig, config.Viper.Get(agent_config.NginxAppProtectMonitoringExtensionPluginConfigKey))
				if err != nil {
					log.Errorf("Unable to load the Nginx App Protect Monitoring plugin due to the following error: %v", err)
				} else {
					extensionPlugins = append(extensionPlugins, nginxAppProtectMonitoringExtensionPlugin)
				}
			default:
				log.Warnf("unknown extension configured: %s", extension)
			}
		}
	}

	return corePlugins, extensionPlugins
}
