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
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 1,
				},
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: getAgentConfig().Command.Server,
				Auth:   getAgentConfig().Command.Auth,
				TLS:    getAgentConfig().Command.TLS,
			},
		},
		{
			name: "Test 2: Missing server",
			protoConfig: &mpi.CommandServer{
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: nil,
				Auth:   getAgentConfig().Command.Auth,
				TLS:    getAgentConfig().Command.TLS,
			},
		},
		{
			name: "Test 3: Missing auth",
			protoConfig: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 1, // gRPC
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
			expected: &Command{
				Server: getAgentConfig().Command.Server,
				Auth:   nil,
				TLS:    getAgentConfig().Command.TLS,
			},
		},
		{
			name: "Test 4: Missing TLS",
			protoConfig: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 1, // Change to HTTP when supported
				},
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
			},
			expected: &Command{
				Server: getAgentConfig().Command.Server,
				Auth:   getAgentConfig().Command.Auth,
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
				Server: getAgentConfig().Command.Server,
				Auth:   getAgentConfig().Command.Auth,
				TLS:    getAgentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 2,
				},
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 2: Missing server",
			cmd: &Command{
				Server: nil,
				Auth:   getAgentConfig().Command.Auth,
				TLS:    getAgentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: nil,
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 3: Missing auth",
			cmd: &Command{
				Server: getAgentConfig().Command.Server,
				Auth:   nil,
				TLS:    getAgentConfig().Command.TLS,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 2, // gRPC
				},
				Tls: &mpi.TLSSettings{
					Cert:       getAgentConfig().Command.TLS.Cert,
					Key:        getAgentConfig().Command.TLS.Key,
					Ca:         getAgentConfig().Command.TLS.Ca,
					ServerName: getAgentConfig().Command.TLS.ServerName,
					SkipVerify: getAgentConfig().Command.TLS.SkipVerify,
				},
			},
		},
		{
			name: "Test 4: Missing TLS",
			cmd: &Command{
				Server: getAgentConfig().Command.Server,
				Auth:   getAgentConfig().Command.Auth,
				TLS:    nil,
			},
			expected: &mpi.CommandServer{
				Server: &mpi.ServerSettings{
					Host: getAgentConfig().Command.Server.Host,
					Port: int32(getAgentConfig().Command.Server.Port),
					Type: 2,
				},
				Auth: &mpi.AuthSettings{
					Token: getAgentConfig().Command.Auth.Token,
				},
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
