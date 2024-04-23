// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const messageID = "964e1e51-44cc-4c55-8422-2a3205bdfc2f"

func CreateProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	newTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		slog.Error("failed to parse time")
		return timestamppb.Now(), err
	}

	return timestamppb.New(newTime), nil
}

func CreateMessageMeta() *v1.MessageMeta {
	return &v1.MessageMeta{
		MessageId:     messageID,
		CorrelationId: correlationID,
		Timestamp:     timestamppb.Now(),
	}
}
