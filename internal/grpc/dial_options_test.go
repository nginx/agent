// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"testing"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"

	"github.com/stretchr/testify/assert"
)

func TestGrpcClient_GetDialOptions(t *testing.T) {
	tests := []struct {
		name        string
		agentConfig *config.Config
		expected    int
	}{
		{
			"Test 1: DialOptions insecure",
			types.GetAgentConfig(),
			7,
		},
		{
			"Test 2: DialOptions TLS",
			types.GetAgentConfig(),
			8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(ttt *testing.T) {
			options := GetDialOptions(tt.agentConfig)
			assert.NotNil(ttt, options)
			assert.Len(ttt, options, tt.expected)
		})
	}
}
