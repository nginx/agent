/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v2/src/core"
)

func TestFileWatcherThrottling(t *testing.T) {
	t.Run("checks that multiple messages works", func(t *testing.T) {
		messages := []*core.Message{
			core.NewMessage(core.DataplaneFilesChanged, 1),
			core.NewMessage("test.message", 2),
			core.NewMessage("test.message", 3),
			core.NewMessage(core.DataplaneFilesChanged, 4),
			core.NewMessage(core.DataplaneFilesChanged, 5),
		}

		fileWatchThrottle := NewFileWatchThrottle()

		messagePipe := core.NewMockMessagePipe(context.Background())

		err := messagePipe.Register(10, []core.Plugin{fileWatchThrottle}, []core.ExtensionPlugin{})
		assert.NoError(t, err)

		defer fileWatchThrottle.Close()

		for _, message := range messages {
			messagePipe.Process(message)
		}

		messagePipe.Run()

		assert.Eventually(t, func() bool { return len(messagePipe.GetMessages()) == 1 }, time.Second*10, time.Second)
	})
}
