// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	agentid "github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateDataPlaneResponse(
	correlationID string,
	commandResponse *mpi.CommandResponse,
	requestType mpi.DataPlaneResponse_RequestType,
	instanceID string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     agentid.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: commandResponse,
		InstanceId:      instanceID,
		RequestType:     requestType,
	}
}
