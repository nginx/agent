// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
)

func TestFromCommandProto(t *testing.T) {
	tests := []struct {
		protoConfig *mpi.CommandServer
		expected    *Command
		name        string
	}{
		{
			name: "Test 1: Valid input with all fields",
			protoConfig: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Auth: &mpi.AuthSettings{},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: agentConfig().Command.Server,
				Auth:   nil,
				TLS:    agentConfig().Command.TLS,
			},
		},
		{
			name: "Test 2: Missing server",
			protoConfig: &mpi.CommandServer{
				Auth: &mpi.AuthSettings{},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: nil,
				Auth:   nil,
				TLS:    agentConfig().Command.TLS,
			},
		},
		{
			name: "Test 3: Missing auth",
			protoConfig: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: agentConfig().Command.Server,
				Auth:   nil,
				TLS:    agentConfig().Command.TLS,
			},
		},
		{
			name: "Test 4: Missing TLS",
			protoConfig: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Auth: &mpi.AuthSettings{},
			},
			expected: &Command{
				Server: agentConfig().Command.Server,
				Auth:   nil,
				TLS:    nil,
			},
		},
		{
			name:        "Test 5: Empty input",
			protoConfig: &mpi.CommandServer{},
			expected: &Command{
				Server: nil,
				Auth:   nil,
				TLS:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := FromCommandProto(tt.protoConfig)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestToCommandProto(t *testing.T) {
	tests := []struct {
		cmd      *Command
		expected *mpi.CommandServer
		name     string
	}{
		{
			name: "Test 1: Valid input with all fields",
			cmd: &Command{
				Server: agentConfig().Command.Server,
				Auth:   agentConfig().Command.Auth,
				TLS:    agentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Auth: &mpi.AuthSettings{},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 2: Missing server",
			cmd: &Command{
				Server: nil,
				Auth:   agentConfig().Command.Auth,
				TLS:    agentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: nil,
				Auth:   &mpi.AuthSettings{},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 3: Missing auth",
			cmd: &Command{
				Server: agentConfig().Command.Server,
				Auth:   nil,
				TLS:    agentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Tls: &mpi.TLSSettings{
					Cert:       agentConfig().Command.TLS.Cert,
					Key:        agentConfig().Command.TLS.Key,
					Ca:         agentConfig().Command.TLS.Ca,
					ServerName: agentConfig().Command.TLS.ServerName,
					SkipVerify: agentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 4: Missing TLS",
			cmd: &Command{
				Server: agentConfig().Command.Server,
				Auth:   agentConfig().Command.Auth,
				TLS:    nil,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: agentConfig().Command.Server.Host,
					Port: int32(agentConfig().Command.Server.Port),
					Type: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
				},
				Auth: &mpi.AuthSettings{},
			},
		},
		{
			name: "Test 5: Empty input",
			cmd:  &Command{},
			expected: &mpi.CommandServer{
				Server: nil,
				Auth:   nil,
				Tls:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protoConfig := ToCommandProto(tt.cmd)
			assert.Equal(t, tt.expected, protoConfig)
		})
	}
}
