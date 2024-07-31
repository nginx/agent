package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	wait "github.com/testcontainers/testcontainers-go/wait"
)

const (
	agentServiceTimeout = 20 * time.Second
	semverRegex         = `v^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-]\d*(?:\.\d*[a-zA-Z-]\d*)*)?))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
)

// SetupTestContainerWithAgent sets up a container with nginx and nginx-agent installed
func SetupTestContainerWithAgent(t *testing.T, testName string, conf string, waitForLog string) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx := context.Background()

	ctxCancel, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	require.NoError(t,
		comp.WaitForService("agent", wait.ForLog(waitForLog)).WithEnv(
			map[string]string{
				"PACKAGE_NAME":                   os.Getenv("PACKAGE_NAME"),
				"PACKAGES_REPO":                  os.Getenv("PACKAGES_REPO"),
				"BASE_IMAGE":                     os.Getenv("BASE_IMAGE"),
				"OS_RELEASE":                     os.Getenv("OS_RELEASE"),
				"OS_VERSION":                     os.Getenv("OS_VERSION"),
				"CONTAINER_OS_TYPE":              os.Getenv("CONTAINER_OS_TYPE"),
				"CONTAINER_NGINX_IMAGE_REGISTRY": os.Getenv("CONTAINER_NGINX_IMAGE_REGISTRY"),
				"TAG":                            os.Getenv("TAG"),
				"CONF_FILE":                      conf,
			},
		).Up(ctxCancel, compose.Wait(true)), "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctxCancel, "agent")
	require.NoError(t, err)

	t.Cleanup(func() {
		logReader, err := testContainer.Logs(ctxCancel)
		assert.NoError(t, err)
		defer logReader.Close()

		testContainerLogs, err := io.ReadAll(logReader)
		assert.NoError(t, err)

		err = os.MkdirAll("/tmp/integration-test-logs/", os.ModePerm)
		assert.NoError(t, err)
		err = os.WriteFile(fmt.Sprintf("/tmp/integration-test-logs/nginx-agent-integration-test-%s.log", testName), testContainerLogs, 0o660)
		assert.NoError(t, err)

		assert.NoError(t, comp.Down(ctxCancel, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	return testContainer
}

// SetupTestContainerWithoutAgent sets up a container with nginx installed
func SetupTestContainerWithoutAgent(t *testing.T) *testcontainers.DockerContainer {
	comp, err := compose.NewDockerCompose(os.Getenv("DOCKER_COMPOSE_FILE"))
	assert.NoError(t, err, "NewDockerComposeAPI()")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err = comp.
		WithEnv(map[string]string{
			"PACKAGE_NAME":                   os.Getenv("PACKAGE_NAME"),
			"PACKAGES_REPO":                  os.Getenv("PACKAGES_REPO"),
			"INSTALL_FROM_REPO":              os.Getenv("INSTALL_FROM_REPO"),
			"BASE_IMAGE":                     os.Getenv("BASE_IMAGE"),
			"OS_RELEASE":                     os.Getenv("OS_RELEASE"),
			"OS_VERSION":                     os.Getenv("OS_VERSION"),
			"CONTAINER_NGINX_IMAGE_REGISTRY": os.Getenv("CONTAINER_NGINX_IMAGE_REGISTRY"),
			"TAG":                            os.Getenv("TAG"),
			"CONTAINER_OS_TYPE":              os.Getenv("CONTAINER_OS_TYPE"),
		}).
		WaitForService("agent", wait.NewLogStrategy("nginx_pid").WithOccurrence(1)).
		Up(ctx, compose.Wait(true))

	assert.NoError(t, err, "compose.Up()")

	testContainer, err := comp.ServiceContainer(ctx, "agent")
	serviceNames := comp.Services()

	assert.Equal(t, 1, len(serviceNames))
	assert.Contains(t, serviceNames, "agent")

	t.Cleanup(func() {
		logReader, err := testContainer.Logs(ctx)
		assert.NoError(t, err)
		defer logReader.Close()

		testContainerLogs, err := io.ReadAll(logReader)
		assert.NoError(t, err)

		err = os.MkdirAll("/tmp/integration-test-logs/", os.ModePerm)
		assert.NoError(t, err)
		err = os.WriteFile("/tmp/integration-test-logs/nginx-agent-integration-test-install-uninstall.log", testContainerLogs, 0o660)
		assert.NoError(t, err)

		assert.NoError(t, comp.Down(ctx, compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
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
	assert.Contains(t, string(agentLogContent), "NGINX Agent v", "agent log file contains invalid agent version")

	semverRe := regexp.MustCompile(semverRegex)

	if semverRe.MatchString(string(agentLogContent)) {
		assert.Fail(t, "failed log content for semver value passed to Agent")
	}

	assert.NotContains(t, string(agentLogContent), "level=error", "agent log file contains logs at error level")
	assert.NotContains(t, string(agentLogContent), "level=panic", "agent log file contains logs at panic level")
	assert.NotContains(t, string(agentLogContent), "level=fatal", "agent log file contains logs at fatal level")
}

func ExecuteCommand(agentContainer *testcontainers.DockerContainer, cmd []string) (string, error) {
	exitCode, response, err := agentContainer.Exec(context.Background(), cmd, tcexec.Multiplexed())
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", errors.New(fmt.Sprintf("Incorrect exit code returned: %d", exitCode))
	}

	responseContent, err := io.ReadAll(response)
	if err != nil {
		return "", err
	}

	return string(responseContent), nil
}
