package features

import (
	"context"
	"io"
	"testing"

	"github.com/nginx/agent/test/integration/utils"
	"github.com/stretchr/testify/assert"
	"os"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	wait "github.com/testcontainers/testcontainers-go/wait"
)

func SetupTestContainerWithAgent(t *testing.T, conf string, waitForLog string) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()
	t.Cleanup(func() {
		assert.NoError(t, comp.Down(ctx, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	require.NoError(t,
		comp.WaitForService("agent", wait.ForLog(waitForLog)).WithEnv(
			map[string]string{
				"PACKAGE_NAME":  os.Getenv("PACKAGE_NAME"),
				"PACKAGES_REPO": os.Getenv("PACKAGES_REPO"),
				"BASE_IMAGE":    os.Getenv("BASE_IMAGE"),
				"OS_RELEASE":    os.Getenv("OS_RELEASE"),
				"OS_VERSION":    os.Getenv("OS_VERSION"),
				"CONF_FILE":     conf,
			},
		).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	require.NoError(t, err)

	return testContainer
}

func TestFeatures_NginxCountingEnabled(t *testing.T) {
	enabledFeatureLogs := []string{"level=info msg=\"NGINX Counter initializing", "level=info msg=\"MetricsThrottle initializing\"", "level=info msg=\"DataPlaneStatus initializing\"",
		"level=info msg=\"OneTimeRegistration initializing\"", "level=info msg=\"Metrics initializing\""}
	disabledFeatureLogs := []string{"level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\""}

	testContainer := SetupTestContainerWithAgent(t, "./test_configs/nginx-agent-counting.conf:/etc/nginx-agent/nginx-agent.conf", "OneTimeRegistration completed")
	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}

}

func TestFeatures_MetricsEnabled(t *testing.T) {
	enabledFeatureLogs := []string{"level=info msg=\"Metrics initializing\"", "level=info msg=\"MetricsThrottle initializing\"", "level=info msg=\"DataPlaneStatus initializing\""}
	disabledFeatureLogs := []string{"level=info msg=\"OneTimeRegistration initializing\"", "level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\""}

	testContainer := SetupTestContainerWithAgent(t, "./test_configs/nginx-agent-metrics.conf:/etc/nginx-agent/nginx-agent.conf", "MetricsThrottle waiting for report ready")
	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}

}

func TestFeatures_ConfigEnabled(t *testing.T) {
	enabledFeatureLogs := []string{"level=info msg=\"DataPlaneStatus initializing\""}
	disabledFeatureLogs := []string{"level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\"", "level=info msg=\"Metrics initializing\"", "level=info msg=\"MetricsThrottle initializing\""}

	testContainer := SetupTestContainerWithAgent(t, "./test_configs/nginx-agent-config.conf:/etc/nginx-agent/nginx-agent.conf", "DataPlaneStatus initializing")
	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}

}
