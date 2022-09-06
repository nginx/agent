package manager

import (
	"context"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSecurityEventManager_Close(t *testing.T) {
	conf := &config.Config{
		Log: config.LogConfig{
			Level: "debug",
		},
		NAPMonitoring: config.NAPMonitoring{
			CollectorBufferSize: 1,
			ProcessorBufferSize: 1,
			SyslogIP:            "0.0.0.0",
			SyslogPort:          1234,
		},
	}

	napMonitoring, err := NewSecurityEventManager(conf)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	go func(ctx context.Context) {
		time.Sleep(5 * time.Second)
		cancel()
	}(ctx)

	napMonitoring.Run(ctx)

	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	}
}
