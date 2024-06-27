// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"path/filepath"
	"testing"

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

func BenchmarkNginxConfigParser_Parse(b *testing.B) {
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			b.Run(configFilePath, func(bb *testing.B) {
				agentConfig.AllowedDirectories = []string{
					filepath.Dir(configFilePath),
				}

				nginxConfigParser := NewNginxConfigParser(
					agentConfig,
				)

				for i := 0; i < bb.N; i++ {
					_, err := nginxConfigParser.Parse(
						ctx,
						&mpi.Instance{
							InstanceMeta: &mpi.InstanceMeta{
								InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX,
							},
							InstanceRuntime: &mpi.InstanceRuntime{
								ConfigPath: configFilePath,
							},
						},
					)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}
