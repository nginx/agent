// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"log/slog"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	newTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		slog.Error("failed to parse time")
		return timestamppb.Now(), err
	}

	return timestamppb.New(newTime), nil
}
