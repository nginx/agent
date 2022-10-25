package manager

import (
	"context"
	"github.com/nginx/agent/v2/src/core/metrics"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
)

func TestManager_Close(t *testing.T) {
	conf := &config.Config{
		Log: config.LogConfig{
			Level: "debug",
		},
		NAPMonitoring: config.NAPMonitoring{
			CollectorBufferSize: 1,
			ProcessorBufferSize: 1,
			SyslogIP:            "127.0.0.1",
			SyslogPort:          1234,
		},
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
