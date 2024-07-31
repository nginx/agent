/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/v2/src/core/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func CreateGrpcClients(ctx context.Context, loadedConfig *config.Config) (client.Controller, client.Commander, client.MetricReporter) {
	if !loadedConfig.IsGrpcServerConfigured() {
		log.Info("GRPC clients not created due to missing server config")
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
	commander.WithBackoffSettings(loadedConfig.GetServerBackoffSettings())

	commander.WithServer(loadedConfig.Server.Target)
	commander.WithDialOptions(append(grpcDialOptions, secureCmdDialOpts)...)

	reporter := client.NewMetricReporterClient()
	reporter.WithBackoffSettings(loadedConfig.GetMetricsBackoffSettings())
	reporter.WithServer(loadedConfig.Server.Target)
	reporter.WithDialOptions(append(grpcDialOptions, secureMetricsDialOpts)...)

	controller.WithClient(commander)
	controller.WithClient(reporter)

	return controller, commander, reporter
}

func setDialOptions(loadedConfig *config.Config) []grpc.DialOption {
	grpcDialOptions := []grpc.DialOption{grpc.WithUserAgent("nginx-agent/" + strings.TrimPrefix(loadedConfig.Version, "v"))}
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(loadedConfig.Server.Token, sdkGRPC.NewMessageMeta(uuid.NewString()))...)
	return grpcDialOptions
}
