// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package proto

import (
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/pkg/uuid"
)

func GenerateMessageID() string {
	uuidv7, err := uuid.GenerateUUIDV7()
	if err != nil {
		slog.Debug("issue generating uuidv7, using sha256 and timestamp instead", "error", err)
		return uuid.Generate("%s", time.Now().String())
	}

	return uuidv7
}
