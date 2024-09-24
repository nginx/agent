// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"log/slog"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// FromCommandProto maps the Protobuf CommandServer message to the AgentConfig struct
func FromCommandProto(config *mpi.CommandServer) *Command {
	cmd := &Command{}
	if config.GetServer() != nil && config.GetServer().GetHost() != "" && config.GetServer().GetPort() != 0 {
		cmd.Server = &ServerConfig{
			Host: config.GetServer().GetHost(),
			Port: int(config.GetServer().GetPort()),
		}
		if config.GetServer().GetType() != mpi.ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED {
			cmd.Server.Type = ServerType(config.GetServer().GetType())
		}
	} else {
		cmd.Server = nil
	}

	if config.GetAuth() != nil && config.GetAuth().GetToken() != "" {
		cmd.Auth = &AuthConfig{
			Token: config.GetAuth().GetToken(),
		}
	} else {
		cmd.Auth = nil
	}

	if config.GetTls() != nil {
		cmd.TLS = &TLSConfig{
			Cert:       config.GetTls().GetCert(),
			Key:        config.GetTls().GetKey(),
			Ca:         config.GetTls().GetCa(),
			ServerName: config.GetTls().GetServerName(),
			SkipVerify: config.GetTls().GetSkipVerify(),
		}
		if cmd.TLS.SkipVerify {
			slog.Warn("SkipVerify is true, this accepts any certificate presented by the server and any host name in that certificate.")
		}
	} else {
		cmd.TLS = nil
	}

	return cmd
}

// ToCommandProto maps the Go Command struct back to the Protobuf CommandServer message
func ToCommandProto(cmd *Command) *mpi.CommandServer {
	protoConfig := &mpi.CommandServer{}

	// Map ServerConfig to the Protobuf ServerConfigProto
	if cmd.Server != nil {
		protoConfig.Server = &mpi.ServerSettings{
			Host: cmd.Server.Host,
			Port: int32(cmd.Server.Port),
			Type: mpi.ServerSettings_ServerType(cmd.Server.Type + 1),
		}
	}

	// Map AuthConfig to the Protobuf AuthConfigProto
	if cmd.Auth != nil {
		protoConfig.Auth = &mpi.AuthSettings{
			Token: cmd.Auth.Token,
		}
	}

	// Map TLSConfig to the Protobuf TLSConfigProto
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
