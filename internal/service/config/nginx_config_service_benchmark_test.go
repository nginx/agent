// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/require"
)

func BenchmarkNginxConfigService_ParseConfig(b *testing.B) {
	nginxConfigService := NewNginx(
		&instances.Instance{
			Type: instances.Type_NGINX,
			Meta: &instances.Meta{
				Meta: &instances.Meta_NginxMeta{
					NginxMeta: &instances.NginxMeta{
						ConfigPath: "../../../test/config/nginx/nginx.conf",
					},
				},
			},
		},
		types.GetAgentConfig(),
	)

	for i := 0; i < b.N; i++ {
		_, err := nginxConfigService.ParseConfig()
		require.NoError(b, err)
	}
}
