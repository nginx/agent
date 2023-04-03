package utils

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	wait "github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestContainerWithAgent sets up a container with nginx and nginx-agent installed
func SetupTestContainerWithAgent(t *testing.T) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()
	t.Cleanup(func() {
		assert.NoError(t, comp.Down(ctx, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	require.NoError(t,
		comp.WaitForService("agent", wait.ForLog("OneTimeRegistration completed")).WithEnv(
			map[string]string{
				"PACKAGE_NAME": os.Getenv("PACKAGE_NAME"),
				"BASE_IMAGE":   os.Getenv("BASE_IMAGE"),
			},
		).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	require.NoError(t, err)

	return testContainer
}

// SetupTestContainerWithoutAgent sets up a container with nginx installed
func SetupTestContainerWithoutAgent(t *testing.T) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()
	t.Cleanup(func() {
		assert.NoError(t, comp.Down(ctx, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	require.NoError(t, comp.WaitForService("agent", wait.ForHTTP("/")).WithEnv(
		map[string]string{
			"PACKAGE_NAME": os.Getenv("PACKAGE_NAME"),
			"BASE_IMAGE":   os.Getenv("BASE_IMAGE"),
		},
	).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	require.NoError(t, err)

	return testContainer
}

func TestAgentHasNoErrorLogs(t *testing.T, agentContainer *testcontainers.DockerContainer) {
	exitCode, agentLogFile, err := agentContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	require.NoError(t, err, "agent log file not found")
	require.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)
	require.NoError(t, err, "agent log file could not be read")

	assert.NotEmpty(t, agentLogContent, "agent log file empty")
	assert.NotContains(t, string(agentLogContent), "level=error", "agent log file contains logs at error level")
	assert.NotContains(t, string(agentLogContent), "level=panic", "agent log file contains logs at panic level")
	assert.NotContains(t, string(agentLogContent), "level=fatal", "agent log file contains logs at fatal level")
}
