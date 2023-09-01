/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"context"
	"os"
	"strconv"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/logger"
	"github.com/nginx/agent/v2/src/extensions"
	"github.com/nginx/agent/v2/src/plugins"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// set at buildtime
	commit  = ""
	version = ""
)

func init() {
	config.InitConfiguration(version, commit)
}

func main() {
	config.RegisterRunner(func(cmd *cobra.Command, _ []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		env := &core.EnvironmentType{}

		loadedConfig, err := config.GetConfig(env.GetSystemUUID())
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		logger.SetLogLevel(loadedConfig.Log.Level)
		logFile := logger.SetLogFile(loadedConfig.Log.Path)
		if logFile != nil {
			defer logFile.Close()
		}

		log.Tracef("Config loaded from disk, %v", loadedConfig)

		if loadedConfig.DisplayName == "" {
			loadedConfig.DisplayName = env.GetHostname()
			log.Infof("setting displayName to %s", loadedConfig.DisplayName)
		}

		log.Infof("NGINX Agent %s at %s with pid %d, clientID=%s name=%s features=%v",
			version, commit, os.Getpid(), loadedConfig.ClientID, loadedConfig.DisplayName, loadedConfig.Features)
		sdkGRPC.InitMeta(loadedConfig.ClientID, loadedConfig.CloudAccountID)

		controller, commander, reporter := core.CreateGrpcClients(ctx, loadedConfig)

		if controller != nil {
			if err := controller.Connect(); err != nil {
				log.Warnf("Unable to connect to control plane: %v", err)
				return
			}
		}

		binary := core.NewNginxBinary(env, loadedConfig)

		corePlugins, extensionPlugins := loadPlugins(commander, binary, env, reporter, loadedConfig)

		pipe := core.InitializePipe(ctx, corePlugins, extensionPlugins, agent_config.DefaultPluginSize)

		event := events.NewAgentEventMeta(config.MODULE,
			version,
			strconv.Itoa(os.Getpid()),
			"Initialize Agent",
			env.GetHostname(),
			env.GetSystemUUID(),
			loadedConfig.InstanceGroup,
			loadedConfig.Tags)

		pipe.Process(core.NewMessage(core.AgentStarted, event))
		core.HandleSignals(ctx, commander, loadedConfig, env, pipe, cancel, controller)

		pipe.Run()
	})

	if err := config.Execute(); err != nil {
		log.Fatal(err)
	}
}

func loadPlugins(commander client.Commander, binary *core.NginxBinaryType, env *core.EnvironmentType, reporter client.MetricReporter, loadedConfig *config.Config) ([]core.Plugin, []core.ExtensionPlugin) {
	var corePlugins []core.Plugin
	var extensionPlugins []core.ExtensionPlugin

	if commander != nil {
		corePlugins = append(corePlugins,
			plugins.NewCommander(commander, loadedConfig),
		)

		if loadedConfig.IsFeatureEnabled(agent_config.FeatureFileWatcher) {
			corePlugins = append(corePlugins,
				plugins.NewFileWatcher(loadedConfig, env),
				plugins.NewFileWatchThrottle(),
			)
		}
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsSender) && reporter != nil {
		corePlugins = append(corePlugins,
			plugins.NewMetricsSender(reporter),
		)
	}

	corePlugins = append(corePlugins,
		plugins.NewConfigReader(loadedConfig),
		plugins.NewNginx(commander, binary, env, loadedConfig),
		plugins.NewExtensions(loadedConfig, env),
		plugins.NewFeatures(commander, loadedConfig, env, binary, version),
	)

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureRegistration) {
		corePlugins = append(corePlugins, plugins.NewOneTimeRegistration(loadedConfig, binary, env, sdkGRPC.NewMessageMeta(uuid.NewString()), version))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsCollection) ||
		(len(loadedConfig.Nginx.NginxCountingSocket) > 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureNginxCounting)) {
		corePlugins = append(corePlugins, plugins.NewMetrics(loadedConfig, env, binary))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureMetrics) || loadedConfig.IsFeatureEnabled(agent_config.FeatureMetricsThrottle) {
		corePlugins = append(corePlugins, plugins.NewMetricsThrottle(loadedConfig, env))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureDataPlaneStatus) {
		corePlugins = append(corePlugins, plugins.NewDataPlaneStatus(loadedConfig, sdkGRPC.NewMessageMeta(uuid.NewString()), binary, env, version))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureProcessWatcher) {
		corePlugins = append(corePlugins, plugins.NewProcessWatcher(env, binary))
	}

	if loadedConfig.IsFeatureEnabled(agent_config.FeatureActivityEvents) {
		corePlugins = append(corePlugins, plugins.NewEvents(loadedConfig, env, sdkGRPC.NewMessageMeta(uuid.NewString()), binary))
	}

	if loadedConfig.AgentAPI.Port != 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureAgentAPI) {
		corePlugins = append(corePlugins, plugins.NewAgentAPI(loadedConfig, env, binary))
	} else {
		log.Info("Agent API not configured")
	}

	if len(loadedConfig.Nginx.NginxCountingSocket) > 0 && loadedConfig.IsFeatureEnabled(agent_config.FeatureNginxCounting) {
		corePlugins = append(corePlugins, plugins.NewNginxCounter(loadedConfig, binary, env))
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
