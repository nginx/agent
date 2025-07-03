// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// FromCommandProto maps the AgentConfig Command struct to the Command proto message
func FromCommandProto(config *mpi.CommandServer) *Command {
	cmd := &Command{}

	// Map ServerSettings to the ServerConfig
	if config.GetServer() != nil && config.GetServer().GetHost() != "" && config.GetServer().GetPort() != 0 {
		cmd.Server = &ServerConfig{
			Host: config.GetServer().GetHost(),
			Port: int(config.GetServer().GetPort()),
		}
		if config.GetServer().GetType() == mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC {
			cmd.Server.Type = Grpc
		}
	} else {
		cmd.Server = nil
	}
	// Set Auth to be nil
	cmd.Auth = nil

	// Map TLSSettings to TLSConfig
	if config.GetTls() != nil {
		cmd.TLS = &TLSConfig{
			Cert:       config.GetTls().GetCert(),
			Key:        config.GetTls().GetKey(),
			Ca:         config.GetTls().GetCa(),
			ServerName: config.GetTls().GetServerName(),
			SkipVerify: config.GetTls().GetSkipVerify(),
		}
		if cmd.TLS.SkipVerify {
			slog.Warn("Insecure setting SkipVerify, this tells the server to accept a certificate with any hostname.")
		}
	} else {
		cmd.TLS = nil
	}

	return cmd
}

// ToCommandProto maps the AgentConfig Command struct back to the Command proto message
func ToCommandProto(cmd *Command) *mpi.CommandServer {
	protoConfig := &mpi.CommandServer{}

	// Map ServerConfig to the ServerSettings
	if cmd.Server != nil {
		protoServerType := mpi.ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED
		if cmd.Server.Type == Grpc {
			protoServerType = mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC
		}

		protoConfig.Server = &mpi.ServerSettings{
			Host: cmd.Server.Host,
			Port: int32(cmd.Server.Port),
			Type: protoServerType,
		}
	}

	// Map AuthConfig to AuthSettings
	if cmd.Auth != nil {
		protoConfig.Auth = &mpi.AuthSettings{}
	}

	// Map TLSConfig to TLSSettings
	if cmd.TLS != nil {
		protoConfig.Tls = &mpi.TLSSettings{
			Cert:       cmd.TLS.Cert,
			Key:        cmd.TLS.Key,
			Ca:         cmd.TLS.Ca,
			ServerName: cmd.TLS.ServerName,
			SkipVerify: cmd.TLS.SkipVerify,
		}
	}

	return protoConfig
}

// ToAuxiliaryCommandServerProto maps the AgentConfig Command struct back to the AuxiliaryCommandServer proto message
func ToAuxiliaryCommandServerProto(cmd *Command) *mpi.AuxiliaryCommandServer {
	protoConfig := &mpi.AuxiliaryCommandServer{}

	// Map ServerConfig to the ServerSettings
	if cmd.Server != nil {
		protoServerType := mpi.ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED
		if cmd.Server.Type == Grpc {
			protoServerType = mpi.ServerSettings_SERVER_SETTINGS_TYPE_GRPC
		}

		protoConfig.Server = &mpi.ServerSettings{
			Host: cmd.Server.Host,
			Port: int32(cmd.Server.Port),
			Type: protoServerType,
		}
	}

	// Map AuthConfig to AuthSettings
	if cmd.Auth != nil {
		protoConfig.Auth = &mpi.AuthSettings{}
	}

	// Map TLSConfig to TLSSettings
	if cmd.TLS != nil {
		protoConfig.Tls = &mpi.TLSSettings{
			Cert:       cmd.TLS.Cert,
			Key:        cmd.TLS.Key,
			Ca:         cmd.TLS.Ca,
			ServerName: cmd.TLS.ServerName,
			SkipVerify: cmd.TLS.SkipVerify,
		}
	}

	return protoConfig
}
