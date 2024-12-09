// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const success = "Success"

func OKDataPlaneResponse() *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     proto.GenerateMessageID(),
			CorrelationId: proto.GenerateMessageID(),
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  mpi.CommandResponse_COMMAND_STATUS_OK,
			Message: success,
		},
		InstanceId: ossInstanceID,
	}
}
