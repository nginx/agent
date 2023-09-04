package core

import (
	"context"
	"testing"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
)

func TestCreateGrpcClients(t *testing.T) {
	loadedConfig := &config.Config{
		TLS: config.TLSConfig{
			Enable:     true,
			SkipVerify: false,
		},
		Server: config.Server{
			GrpcPort: 6789,
			Host: "192.0.2.4",
		},
	}

	ctx := context.Background()

	controller, commander, reporter := CreateGrpcClients(ctx, loadedConfig)

	// Assert that the returned clients are not nil
	assert.NotNil(t, controller)
	assert.NotNil(t, commander)
	assert.NotNil(t, reporter)
}

func TestSetDialOptions(t *testing.T) {
	loadedConfig := &config.Config{
		TLS: config.TLSConfig{
			Enable:     true,
			SkipVerify: false,
		},
		Server: config.Server{
			GrpcPort: 67890,
			Host: "192.0.2.5",
		},
	}

	dialOptions := setDialOptions(loadedConfig)

	assert.NotEmpty(t, dialOptions)
}
