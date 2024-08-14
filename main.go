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

	"github.com/nginx/agent/sdk/v2/agent/events"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/logger"
	"github.com/nginx/agent/v2/src/plugins"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// set at buildtime
	commit  = ""
	version = ""
	env     = &core.EnvironmentType{}
)

func main() {
	config.InitFlags(version, commit)

	defer func() {
		if err := recover(); err != nil {
			log.Printf("Recovered: %v", err)
		}
	}()

	config.RegisterRunner(func(cmd *cobra.Command, _ []string) {
		config.InitConfigurationFiles()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		loadedConfig, err := config.GetConfig(env.GetSystemUUID())
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		eventMeta := events.NewAgentEventMeta(
			config.MODULE,
			version,
			strconv.Itoa(os.Getpid()),
			env.GetHostname(),
			env.GetSystemUUID(),
			loadedConfig.InstanceGroup,
			loadedConfig.Tags)

		logger.SetLogLevel(loadedConfig.Log.Level)
		logFile := logger.SetLogFile(loadedConfig.Log.Path)
		if logFile != nil {
			defer logFile.Close()
		}

		if config.MigratedEnv {
			log.Warnf("The environment variable prefix 'NMS' is deprecated. Prefix has been migrated to 'NGINX_AGENT'. Please update your configuration to use the new prefix.")
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
			go controller.Connect()
		}

		binary := core.NewNginxBinary(env, loadedConfig)

		corePlugins, extensionPlugins := plugins.LoadPlugins(commander, binary, env, reporter, loadedConfig, eventMeta)

		pipe := core.InitializePipe(ctx, corePlugins, extensionPlugins, loadedConfig.QueueSize)
		pipe.Process(core.NewMessage(core.AgentStarted, eventMeta))
		core.HandleSignals(ctx, commander, loadedConfig, env, pipe, cancel, controller)

		pipe.Run()
	})

	if err := config.Execute(); err != nil {
		log.Fatal(err)
	}
}
