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
	"runtime"
	"strconv"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/logger"
	"github.com/nginx/agent/v2/src/plugins"

	"github.com/grafana/pyroscope-go"
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

		runtime.SetMutexProfileFraction(5)
		runtime.SetBlockProfileRate(5)

		_, err = pyroscope.Start(pyroscope.Config{
			ApplicationName: "nginx.agent",

			// replace this with the address of pyroscope server
			ServerAddress: "http://pyroscope:4040",

			// you can disable logging by setting this to nil
			Logger: pyroscope.StandardLogger,

			// you can provide static tags via a map:
			Tags: map[string]string{"hostname": os.Getenv("HOSTNAME")},

			ProfileTypes: []pyroscope.ProfileType{
				// these profile types are enabled by default:
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,

				// these profile types are optional:
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		})

		if err != nil {
			log.Errorf("Could not start profiler: %v", err)
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

		corePlugins, extensionPlugins := plugins.LoadPlugins(commander, binary, env, reporter, loadedConfig)

		pipe := core.InitializePipe(ctx, corePlugins, extensionPlugins, agent_config.DefaultPluginSize)

		event := events.NewAgentEventMeta(
			config.MODULE,
			version,
			strconv.Itoa(os.Getpid()),
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
