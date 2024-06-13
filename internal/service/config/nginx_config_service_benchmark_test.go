// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"log/slog"
	"testing"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"

	"github.com/nginx/agent/v3/internal/client/clientfakes"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/require"
)

var configFilePaths = []string{
	"../../../test/config/nginx/nginx.conf",
	"../../../test/config/nginx/nginx-with-1k-lines.conf",
	"../../../test/config/nginx/nginx-with-2k-lines.conf",
	"../../../test/config/nginx/nginx-with-3k-lines.conf",
	"../../../test/config/nginx/nginx-with-10k-lines.conf",
}

func BenchmarkNginxConfigService_ParseConfig(b *testing.B) {
	ctx := context.Background()

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			b.Run(configFilePath, func(bb *testing.B) {
				slogger := logger.New(config.Log{Level: "error"})
				slog.SetDefault(slogger)

				nginxConfigService := NewNginx(
					ctx,
					&mpi.Instance{
						InstanceMeta: &mpi.InstanceMeta{
							InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX,
						},
						InstanceRuntime: &mpi.InstanceRuntime{
							ConfigPath: configFilePath,
						},
					},
					types.AgentConfig(),
					&clientfakes.FakeConfigClient{},
				)

				for i := 0; i < bb.N; i++ {
					_, err := nginxConfigService.ParseConfig(ctx)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}
