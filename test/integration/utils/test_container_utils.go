package utils

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	wait "github.com/testcontainers/testcontainers-go/wait"
)

const agentServiceTimeout = 20 * time.Second

// SetupTestContainerWithAgent sets up a container with nginx and nginx-agent installed
func SetupTestContainerWithAgent(t *testing.T) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	require.NoError(t,
		comp.WaitForService("agent", wait.ForLog("OneTimeRegistration completed").WithStartupTimeout(agentServiceTimeout)).WithEnv(
			map[string]string{
				"PACKAGE_NAME":  os.Getenv("PACKAGE_NAME"),
				"PACKAGES_REPO": os.Getenv("PACKAGES_REPO"),
				"BASE_IMAGE":    os.Getenv("BASE_IMAGE"),
				"OS_RELEASE":    os.Getenv("OS_RELEASE"),
				"OS_VERSION":    os.Getenv("OS_VERSION"),
			},
		).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	assert.NoError(t, err)

	t.Cleanup(func() {
		logReader, err := testContainer.Logs(ctxCancel)
		assert.NoError(t, err)
		defer logReader.Close()

		testContainerLogs, err := io.ReadAll(logReader)
		assert.NoError(t, err)

		err = os.WriteFile("/tmp/nginx-agent/integration-test/api.log", testContainerLogs, 0660)
		assert.NoError(t, err)
		assert.NoError(t, comp.Down(ctxCancel, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	return testContainer
}

// SetupTestContainerWithoutAgent sets up a container with nginx installed
func SetupTestContainerWithoutAgent(t *testing.T) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	assert.NoError(t, comp.WaitForService("agent", wait.ForHTTP("/").WithStartupTimeout(agentServiceTimeout)).WithEnv(
		map[string]string{
			"PACKAGE_NAME":      os.Getenv("PACKAGE_NAME"),
			"PACKAGES_REPO":     os.Getenv("PACKAGES_REPO"),
			"INSTALL_FROM_REPO": os.Getenv("INSTALL_FROM_REPO"),
			"BASE_IMAGE":        os.Getenv("BASE_IMAGE"),
			"OS_RELEASE":        os.Getenv("OS_RELEASE"),
			"OS_VERSION":        os.Getenv("OS_VERSION"),
		},
	).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	assert.NoError(t, err)

	t.Cleanup(func() {
		logReader, err := testContainer.Logs(ctxCancel)
		assert.NoError(t, err)
		defer logReader.Close()

		testContainerLogs, err := io.ReadAll(logReader)
		assert.NoError(t, err)

		log.Info("Writing install/uninstall test log file")

		err = os.WriteFile("/tmp/nginx-agent/integration-test/install-uninstall.log", testContainerLogs, 0660)
		assert.NoError(t, err)

		assert.NoError(t, comp.Down(ctxCancel, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

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
