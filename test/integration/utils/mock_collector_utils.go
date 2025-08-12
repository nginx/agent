package utils

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
)

var MockCollectorStack *helpers.MockCollectorContainers

// SetupMetricsTest similar to SetupConnectionTest()
func SetupMetricsTest(tb testing.TB) func(testing.TB) {
	tb.Helper()
	ctx := context.Background()

	if os.Getenv("TEST_ENV") == "Container" {
		setupStackEnvironment(ctx, tb)
	}
	return func(tb testing.TB) {
		tb.Helper()

		if os.Getenv("TEST_ENV") == "Container" {
			helpers.LogAndTerminateStack(
				ctx,
				tb,
				MockCollectorStack,
			)
		}
	}
}

func setupStackEnvironment(ctx context.Context, tb testing.TB) {
	tb.Helper()
	tb.Log("Running tests in a container environment")

	containerNetwork := createContainerNetwork(ctx, tb)
	setupMockCollectorStack(ctx, tb, containerNetwork)
}

func setupMockCollectorStack(ctx context.Context, tb testing.TB, containerNetwork *testcontainers.DockerNetwork) {
	tb.Helper()

	tb.Log("Starting mock collector stack")

	agentConfig := "../../mock/collector/nginx-agent.conf"
	MockCollectorStack = helpers.StartMockCollectorStack(ctx, tb, containerNetwork, agentConfig)
}
