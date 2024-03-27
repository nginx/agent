// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
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
	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			b.Run(configFilePath, func(bb *testing.B) {
				nginxConfigService := NewNginx(
					&v1.Instance{
						InstanceMeta: &v1.InstanceMeta{
							InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
						},
						InstanceConfig: &v1.InstanceConfig{
							Config: &v1.InstanceConfig_NginxConfig{
								NginxConfig: &v1.NGINXConfig{
									ConfigPath: configFilePath,
								},
							},
						},
					},
					types.GetAgentConfig(),
				)

				for i := 0; i < bb.N; i++ {
					_, err := nginxConfigService.ParseConfig()
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}
