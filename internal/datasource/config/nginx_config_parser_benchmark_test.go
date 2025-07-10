// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
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
	// Discard log messages
	slog.SetDefault(slog.New(slog.DiscardHandler))
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

				bb.ResetTimer()

				for range bb.N {
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

// These tests don't exercise the traversal very well, they are more to track the growth of configs in size
func BenchmarkNginxConfigParserGeneratedConfig_Parse(b *testing.B) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	tests := []struct {
		name     string
		fileSize int64
	}{
		{
			name:     "100 KB",
			fileSize: int64(1 * 1024 * 1024 / 10), // 100 KB
		},
		{
			name:     "1 MB",
			fileSize: int64(1 * 1024 * 1024), // 1 MB
		},
		{
			name:     "10 MB",
			fileSize: int64(10 * 1024 * 1024), // 10 MB
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(bb *testing.B) {
			location := bb.TempDir()
			fileName := fmt.Sprintf("%s/%d_%s", location, test.fileSize, "nginx.conf")

			_, err := helpers.GenerateConfig(bb, fileName, test.fileSize)
			require.NoError(b, err)

			agentConfig.AllowedDirectories = []string{
				location,
			}

			nginxConfigParser := NewNginxConfigParser(
				agentConfig,
			)

			bb.ResetTimer()

			for range bb.N {
				_, parseErr := nginxConfigParser.Parse(
					ctx,
					&mpi.Instance{
						InstanceMeta: &mpi.InstanceMeta{
							InstanceType: mpi.InstanceMeta_INSTANCE_TYPE_NGINX,
						},
						InstanceRuntime: &mpi.InstanceRuntime{
							ConfigPath: location,
						},
					},
				)
				require.NoError(bb, parseErr)
			}
		})
	}
}
