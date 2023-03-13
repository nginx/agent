/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package manager

import (
	"context"
	"github.com/nginx/agent/v2/src/core/metrics"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_Close(t *testing.T) {
	conf := &NginxAppProtectMonitoringConfig{
		CollectorBufferSize: 1,
		ProcessorBufferSize: 1,
		SyslogIP:            "127.0.0.1",
		SyslogPort:          1235,
	}

	m, err := NewManager(conf, &metrics.CommonDim{})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		time.Sleep(1 * time.Second)
		cancel()
	}(ctx)

	m.Run(ctx)

	assert.Equal(t, context.Canceled, ctx.Err())
}
