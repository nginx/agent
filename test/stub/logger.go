// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package stub

import (
	"bytes"
	"log/slog"
)

// StubLoggerWith follows the pattern for replacing slog
// with a handler that can take a buffer. The buffer gets filled
// by adding log statements to it as the code is executed.
// You can see more information in the following video for context
// https://www.youtube.com/watch?v=i1bDIyIaxbE

func StubLoggerWith(buffer *bytes.Buffer) {
	mockLoggerHandler := slog.NewTextHandler(buffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(mockLoggerHandler)
	slog.SetDefault(logger)
}
