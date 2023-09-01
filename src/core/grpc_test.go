package core

import (
	"context"
	"testing"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
)

func TestCreateGrpcClients(t *testing.T) {
	// Create a mock configuration with proper settings
	loadedConfig := &config.Config{
		TLS: config.TLSConfig{
			Enable:     true,
			Cert:       "cert.pem",
			Key:        "key.pem",
			Ca:         "ca.pem",
			SkipVerify: false,
		},
		Server: config.Server{
			Metrics: "metrics-server",
			Command: "command-server",
			Target:  "grpc-server",
		},
	}

	ctx := context.Background()

	controller, commander, reporter := CreateGrpcClients(ctx, loadedConfig)

	// Assert that the returned clients are not nil
	assert.NotNil(t, controller)
	assert.NotNil(t, commander)
	assert.NotNil(t, reporter)

	// Additional assertions can be added to test the client configurations
	// For example, you can check if the dial options are set correctly.
}

func TestSetDialOptions(t *testing.T) {
	// Create a mock configuration with proper settings
	loadedConfig := &config.Config{
		Server: config.Server{
			Token: "your-token",
		},
	}

	dialOptions := setDialOptions(loadedConfig)

	// Assert that the dial options are not empty and contain expected options.
	assert.NotEmpty(t, dialOptions)

	// You can add more specific assertions based on your expected dial options.
	// For example, checking if grpc.WithUserAgent is set, or other options.
}
