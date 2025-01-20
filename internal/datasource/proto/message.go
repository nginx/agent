// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"log/slog"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	agentUuid "github.com/nginx/agent/v3/pkg/uuid"
)

// UUIDGenerator defines a function type for generating UUIDs.
type UUIDGenerator func() (uuid.UUID, error)

// DefaultUUIDGenerator is the production implementation for generating UUIDv7.
var defaultUUIDGenerator UUIDGenerator = uuid.NewUUID

// GenerateMessageID generates a unique message ID, falling back to sha256 and timestamp if UUID generation fails.
func GenerateMessageID() string {
	uuidv7, err := defaultUUIDGenerator()
	if err != nil {
		slog.Debug("Issue generating uuidv7, using sha256 and timestamp instead", "error", err)
		return agentUuid.Generate("%s", time.Now().String())
	}

	return uuidv7.String()
}

func CreateDataPlaneResponse(correlationID string, status mpi.CommandResponse_CommandStatus,
	message, instanceID, err string,
) *mpi.DataPlaneResponse {
	return &mpi.DataPlaneResponse{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		CommandResponse: &mpi.CommandResponse{
			Status:  status,
			Message: message,
			Error:   err,
		},
		InstanceId: instanceID,
	}
}
