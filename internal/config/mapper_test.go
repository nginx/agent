// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapper_FromCommandProto(t *testing.T) {
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
			t.Parallel()
			config := FromCommandProto(tt.protoConfig)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestMapper_ToCommandProto(t *testing.T) {
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
			t.Parallel()
			protoConfig := ToCommandProto(tt.cmd)
			assert.Equal(t, tt.expected, protoConfig)
		})
	}
}

func TestMapper_ToAgentConfigLogProto(t *testing.T) {
	tests := []struct {
		log      *Log
		expected *mpi.Log
		name     string
	}{
		{
			name: "Test 1: Log level DEBUG",
			log: &Log{
				Level: "DEBUG",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_DEBUG,
				LogPath:  "",
			},
		},
		{
			name: "Test 2: Log level INFO",
			log: &Log{
				Level: "INFO",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_INFO,
				LogPath:  "",
			},
		},
		{
			name: "Test 3: Log level WARN",
			log: &Log{
				Level: "WARN",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_WARN,
				LogPath:  "",
			},
		},
		{
			name: "Test 4: Log level ERROR",
			log: &Log{
				Level: "ERROR",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_ERROR,
				LogPath:  "",
			},
		},
		{
			name: "Test 5: Log path set",
			log: &Log{
				Level: "INFO",
				Path:  "/path/to/agent.log",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_INFO,
				LogPath:  "/path/to/agent.log",
			},
		},
		{
			name: "Test 6: Both log level and path set",
			log: &Log{
				Level: "DEBUG",
				Path:  "/other/path/to/agent.log",
			},
			expected: &mpi.Log{
				LogLevel: mpi.Log_LOG_LEVEL_DEBUG,
				LogPath:  "/other/path/to/agent.log",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			protoLog := ToAgentConfigLogProto(testCase.log)
			assert.Equal(t, testCase.expected.GetLogLevel(), protoLog.GetLogLevel())
			assert.Equal(t, testCase.expected.GetLogPath(), protoLog.GetLogPath())
		})
	}
}

func TestMapper_ToAuxiliaryCommandServerProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmd      *Command
		wantHost string
		wantPort int32
		wantType mpi.ServerSettings_ServerType
		wantTLS  bool
		wantAuth bool
	}{
		{
			name: "Test 1: full command with gRPC server, TLS and auth",
			cmd: &Command{
				Server: agentConfig().Command.Server,
				TLS:    agentConfig().Command.TLS,
				Auth:   agentConfig().Command.Auth,
			},
			wantHost: "127.0.0.1",
			wantPort: 8888,
			wantType: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
			wantTLS:  true,
			wantAuth: true,
		},
		{
			name:    "Test 2: nil server produces nil server field",
			cmd:     &Command{Server: nil},
			wantTLS: false,
		},
		{
			name: "Test 3: undefined server type maps to UNDEFINED",
			cmd: &Command{
				Server: &ServerConfig{Host: "host", Port: 9090, Type: ""},
			},
			wantHost: "host",
			wantPort: 9090,
			wantType: mpi.ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED,
		},
		{
			name: "Test 4: nil auth produces nil auth field",
			cmd: &Command{
				Server: agentConfig().Command.Server,
				Auth:   nil,
			},
			wantHost: "127.0.0.1",
			wantPort: 8888,
			wantType: mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC,
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ToAuxiliaryCommandServerProto(tt.cmd)
			require.NotNil(t, result)
			if tt.cmd.Server != nil {
				assert.Equal(t, tt.wantHost, result.GetServer().GetHost())
				assert.Equal(t, tt.wantPort, result.GetServer().GetPort())
				assert.Equal(t, tt.wantType, result.GetServer().GetType())
			} else {
				assert.Nil(t, result.GetServer())
			}
			assert.Equal(t, tt.wantTLS, result.GetTls() != nil)
			assert.Equal(t, tt.wantAuth, result.GetAuth() != nil)
		})
	}
}

func TestMapper_FromAgentRemoteConfigProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         *mpi.AgentConfig
		wantLogLevel  string
		wantLogPath   string
		wantLabelKeys []string
	}{
		{
			name: "Test 1: log level and path populated",
			input: &mpi.AgentConfig{
				Log: &mpi.Log{
					LogLevel: mpi.Log_LOG_LEVEL_DEBUG,
					LogPath:  "/var/log/agent.log",
				},
			},
			wantLogLevel: "DEBUG",
			wantLogPath:  "/var/log/agent.log",
		},
		{
			name:  "Test 2: nil log produces nil Log field",
			input: &mpi.AgentConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := FromAgentRemoteConfigProto(tt.input)
			require.NotNil(t, result)
			if tt.input.GetLog() != nil {
				require.NotNil(t, result.Log)
				assert.Equal(t, tt.wantLogLevel, result.Log.Level)
				assert.Equal(t, tt.wantLogPath, result.Log.Path)
			} else {
				assert.Nil(t, result.Log)
			}
		})
	}
}

func TestMapper_MapConfigLogLevelToSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		input    mpi.Log_LogLevel
	}{
		{"Test 1: debug level", "DEBUG", mpi.Log_LOG_LEVEL_DEBUG},
		{"Test 2: info level", "INFO", mpi.Log_LOG_LEVEL_INFO},
		{"Test 3: warn level", "WARN", mpi.Log_LOG_LEVEL_WARN},
		{"Test 4: error level", "ERROR", mpi.Log_LOG_LEVEL_ERROR},
		{"Test 5: unrecognized value falls back to INFO", "INFO", mpi.Log_LogLevel(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, MapConfigLogLevelToSlogLevel(tt.input))
		})
	}
}

func TestMapper_MapSlogLevelToConfigLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected mpi.Log_LogLevel
	}{
		{"Test 1: debug input is case-insensitive", "debug", mpi.Log_LOG_LEVEL_DEBUG},
		{"Test 2: unknown input falls back to INFO", "unknown", mpi.Log_LOG_LEVEL_INFO},
		{"Test 3: empty input falls back to INFO", "", mpi.Log_LOG_LEVEL_INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, MapSlogLevelToConfigLogLevel(tt.input))
		})
	}
}
