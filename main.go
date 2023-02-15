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
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/logger"
	"github.com/nginx/agent/v2/src/extensions"
	"github.com/nginx/agent/v2/src/plugins"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
)

var (
	// set at buildtime
	commit  = ""
	version = ""
)

func init() {
	config.SetVersion(version, commit)
	config.SetDefaults()
	config.RegisterFlags()
	configPath, err := config.RegisterConfigFile(config.DynamicConfigFileAbsPath, config.ConfigFileName, config.ConfigFilePaths()...)
	if err != nil {
		log.Fatalf("Failed to load configuration file: %v", err)
	}
	log.Debugf("Configuration file loaded %v", configPath)
	config.Viper.Set(config.ConfigPathKey, configPath)
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
		if loadedConfig.DisplayName == "" {
			loadedConfig.DisplayName = env.GetHostname()
			log.Infof("setting displayName to %s", loadedConfig.DisplayName)
		}

		log.Infof("NGINX Agent %s at %s with pid %d, clientID=%s name=%s features=%v",
			version, commit, os.Getpid(), loadedConfig.ClientID, loadedConfig.DisplayName, loadedConfig.Features)
		sdkGRPC.InitMeta(loadedConfig.ClientID, loadedConfig.CloudAccountID)

		controller, commander, reporter := createGrpcClients(ctx, loadedConfig)

		if controller != nil {
			if err := controller.Connect(); err != nil {
				log.Warnf("Unable to connect to control plane: %v", err)
				return
			}
		}

		binary := core.NewNginxBinary(env, loadedConfig)

		corePlugins, extensionPlugins := loadPlugins(commander, binary, env, reporter, loadedConfig)

		pipe := initializeMessagePipe(ctx, corePlugins, extensionPlugins)

		pipe.Process(core.NewMessage(core.AgentStarted,
			plugins.NewAgentEventMeta(version, strconv.Itoa(os.Getpid()))),
		)

		handleSignals(ctx, commander, loadedConfig, env, pipe, cancel)

		pipe.Run()
	})

	if err := config.Execute(); err != nil {
		log.Fatal(err)
	}
}

// handleSignals handles signals to attempt graceful shutdown
// for now it also handles sending the agent stopped event because as of today we don't have a mechanism for synchronizing
// tasks between multiple plugins from outside a plugin
func handleSignals(
	ctx context.Context,
	cmder client.Commander,
	loadedConfig *config.Config,
	env core.Environment,
	pipe core.MessagePipeInterface,
	cancel context.CancelFunc,
) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			stopCmd := plugins.GenerateAgentStopEventCommand(
				plugins.NewAgentEventMeta(version, strconv.Itoa(os.Getpid())), loadedConfig, env,
			)
			log.Debugf("Sending agent stopped event: %v", stopCmd)

			if err := cmder.Send(ctx, client.MessageFromCommand(stopCmd)); err != nil {
				log.Errorf("Error sending AgentStopped event to command channel: %v", err)
			}

			log.Warn("NGINX Agent exiting")
			cancel()

			timeout := time.Second * 5
			time.Sleep(timeout)
			log.Fatalf("Failed to gracefully shutdown within timeout of %v. Exiting", timeout)
		case <-ctx.Done():
		}
	}()
}

func createGrpcClients(ctx context.Context, loadedConfig *config.Config) (client.Controller, client.Commander, client.MetricReporter) {
	if !loadedConfig.IsGrpcServerConfigured() {
		log.Infof("GRPC clients not created")
		return nil, nil, nil
	}

	grpcDialOptions := setDialOptions(loadedConfig)
	secureMetricsDialOpts, err := sdkGRPC.SecureDialOptions(
		loadedConfig.TLS.Enable,
		loadedConfig.TLS.Cert,
		loadedConfig.TLS.Key,
		loadedConfig.TLS.Ca,
		loadedConfig.Server.Metrics,
		loadedConfig.TLS.SkipVerify)
	if err != nil {
		log.Fatalf("Failed to load secure metric gRPC dial options: %v", err)
	}

	secureCmdDialOpts, err := sdkGRPC.SecureDialOptions(
		loadedConfig.TLS.Enable,
		loadedConfig.TLS.Cert,
		loadedConfig.TLS.Key,
		loadedConfig.TLS.Ca,
		loadedConfig.Server.Command,
		loadedConfig.TLS.SkipVerify)
	if err != nil {
		log.Fatalf("Failed to load secure command gRPC dial options: %v", err)
	}

	controller := client.NewClientController()
	controller.WithContext(ctx)
	commander := client.NewCommanderClient()

	commander.WithServer(loadedConfig.Server.Target)
	commander.WithDialOptions(append(grpcDialOptions, secureCmdDialOpts)...)

	reporter := client.NewMetricReporterClient()
	reporter.WithServer(loadedConfig.Server.Target)
	reporter.WithDialOptions(append(grpcDialOptions, secureMetricsDialOpts)...)

	controller.WithClient(commander)
	controller.WithClient(reporter)

	return controller, commander, reporter
}

func loadPlugins(commander client.Commander, binary *core.NginxBinaryType, env *core.EnvironmentType, reporter client.MetricReporter, loadedConfig *config.Config) ([]core.Plugin, []core.ExtensionPlugin) {
	var corePlugins []core.Plugin
	var extensionPlugins []core.ExtensionPlugin

	if commander != nil {
		corePlugins = append(corePlugins,
			plugins.NewCommander(commander, loadedConfig),
			plugins.NewFileWatcher(loadedConfig, env),
			plugins.NewFileWatchThrottle(),
		)
	}

	if reporter != nil {
		corePlugins = append(corePlugins,
			plugins.NewMetricsSender(reporter),
		)
	}

	corePlugins = append(corePlugins,
		plugins.NewConfigReader(loadedConfig),
		plugins.NewNginx(commander, binary, env, loadedConfig),
		plugins.NewOneTimeRegistration(loadedConfig, binary, env, sdkGRPC.NewMessageMeta(uuid.NewString()), version),
		plugins.NewMetrics(loadedConfig, env, binary),
		plugins.NewMetricsThrottle(loadedConfig, env),
		plugins.NewDataPlaneStatus(loadedConfig, sdkGRPC.NewMessageMeta(uuid.NewString()), binary, env, version),
		plugins.NewProcessWatcher(env, binary),
		plugins.NewExtensions(loadedConfig, env),
		plugins.NewEvents(loadedConfig, env, sdkGRPC.NewMessageMeta(uuid.NewString()), binary),
	)

	if loadedConfig.AgentAPI.Port != 0 {
		corePlugins = append(corePlugins, plugins.NewAgentAPI(loadedConfig, env, binary))
	} else {
		log.Info("Agent API not configured")
	}

	if len(loadedConfig.Nginx.NginxCountingSocket) > 0 {
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

func initializeMessagePipe(ctx context.Context, corePlugins []core.Plugin, extensionPlugins []core.ExtensionPlugin) core.MessagePipeInterface {
	pipe := core.NewMessagePipe(ctx)
	err := pipe.Register(agent_config.DefaultPluginSize, corePlugins, extensionPlugins)
	if err != nil {
		log.Warnf("Failed to start agent successfully, error loading plugins %v", err)
	}
	return pipe
}

func setDialOptions(loadedConfig *config.Config) []grpc.DialOption {
	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(loadedConfig.Server.Token, sdkGRPC.NewMessageMeta(uuid.NewString()))...)
	return grpcDialOptions
}
