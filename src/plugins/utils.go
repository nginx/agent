/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
)

func newStatusCommand(cmd *proto.Command) *proto.Command {
	c := &proto.Command{
		Meta: grpc.NewMessageMeta(uuid.New().String()),
	}
	if cmd != nil {
		c.Meta.MessageId = cmd.Meta.MessageId
	}
	return c
}

func newOKStatus(message string) *proto.Command_CmdStatus {
	return &proto.Command_CmdStatus{
		CmdStatus: &proto.CommandStatusResponse{
			Status:  proto.CommandStatusResponse_CMD_OK,
			Message: message,
		},
	}
}

func newErrStatus(message string) *proto.Command_CmdStatus {
	return &proto.Command_CmdStatus{
		CmdStatus: &proto.CommandStatusResponse{
			Status:  proto.CommandStatusResponse_CMD_ERROR,
			Message: message,
			Error:   message,
		},
	}
}
