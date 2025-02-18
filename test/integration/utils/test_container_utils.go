package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	configFilePermissions = 0o700
	semverRegex           = `v^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-]\d*(?:\.\d*[a-zA-Z-]\d*)*)?))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`
)

type Parameters struct {
	NginxConfigPath      string
	NginxAgentConfigPath string
	LogMessage           string
}

func StartContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	parameters *Parameters,
) testcontainers.Container {
	tb.Helper()

	packageName := Env(tb, "PACKAGE_NAME")
	packageRepo := Env(tb, "PACKAGES_REPO")
	baseImage := Env(tb, "BASE_IMAGE")
	osRelease := Env(tb, "OS_RELEASE")
	osVersion := Env(tb, "OS_VERSION")
	buildTarget := Env(tb, "BUILD_TARGET")
	dockerfilePath := Env(tb, "DOCKERFILE_PATH")
	containerRegistry := Env(tb, "CONTAINER_NGINX_IMAGE_REGISTRY")
	tag := Env(tb, "TAG")
	imagePath := Env(tb, "IMAGE_PATH")
	containerOsType := Env(tb, "CONTAINER_OS_TYPE")

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../../",
			Dockerfile:    dockerfilePath,
			KeepImage:     false,
			PrintBuildLog: true,
			BuildArgs: map[string]*string{
				"PACKAGE_NAME":                   ToPtr(packageName),
				"PACKAGES_REPO":                  ToPtr(packageRepo),
				"BASE_IMAGE":                     ToPtr(baseImage),
				"OS_RELEASE":                     ToPtr(osRelease),
				"OS_VERSION":                     ToPtr(osVersion),
				"ENTRY_POINT":                    ToPtr("./test/docker/entrypoint.sh"),
				"CONTAINER_NGINX_IMAGE_REGISTRY": ToPtr(containerRegistry),
				"IMAGE_PATH":                     ToPtr(imagePath),
				"TAG":                            ToPtr(tag),
				"CONTAINER_OS_TYPE":              ToPtr(containerOsType),
			},
			BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
				buildOptions.Target = buildTarget
			},
		},
		ExposedPorts: []string{"9091/tcp"},
		WaitingFor:   wait.ForLog(parameters.LogMessage),
		Networks: []string{
			containerNetwork.Name,
		},
		NetworkAliases: map[string][]string{
			containerNetwork.Name: {
				"agent",
			},
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      parameters.NginxAgentConfigPath,
				ContainerFilePath: "/etc/nginx-agent/nginx-agent.conf",
				FileMode:          configFilePermissions,
			},
		},
	}

	if parameters.NginxConfigPath != "" {
		req.Files = append(req.Files, testcontainers.ContainerFile{
			HostFilePath:      parameters.NginxConfigPath,
			ContainerFilePath: "/etc/nginx/nginx.conf",
			FileMode:          configFilePermissions,
		})
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(tb, err)

	return container
}

func StartAgentlessContainer(
	ctx context.Context,
	tb testing.TB,
	parameters *Parameters,
) testcontainers.Container {
	tb.Helper()

	packageName := Env(tb, "PACKAGE_NAME")
	packageRepo := Env(tb, "PACKAGES_REPO")
	baseImage := Env(tb, "BASE_IMAGE")
	osRelease := Env(tb, "OS_RELEASE")
	osVersion := Env(tb, "OS_VERSION")
	dockerfilePath := Env(tb, "DOCKERFILE_PATH")
	containerRegistry := Env(tb, "CONTAINER_NGINX_IMAGE_REGISTRY")
	tag := Env(tb, "TAG")
	imagePath := Env(tb, "IMAGE_PATH")

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../../",
			Dockerfile:    dockerfilePath,
			KeepImage:     false,
			PrintBuildLog: true,
			BuildArgs: map[string]*string{
				"PACKAGE_NAME":                   ToPtr(packageName),
				"PACKAGES_REPO":                  ToPtr(packageRepo),
				"BASE_IMAGE":                     ToPtr(baseImage),
				"OS_RELEASE":                     ToPtr(osRelease),
				"OS_VERSION":                     ToPtr(osVersion),
				"ENTRY_POINT":                    ToPtr("./test/docker/agentless-entrypoint.sh"),
				"CONTAINER_NGINX_IMAGE_REGISTRY": ToPtr(containerRegistry),
				"IMAGE_PATH":                     ToPtr(imagePath),
				"TAG":                            ToPtr(tag),
			},
			BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
				buildOptions.Target = "install-nginx"
			},
		},
		ExposedPorts: []string{"9091/tcp"},
		WaitingFor:   wait.ForLog(parameters.LogMessage),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      parameters.NginxConfigPath,
				ContainerFilePath: "/etc/nginx/nginx.conf",
				FileMode:          configFilePermissions,
			},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(tb, err)

	return container
}

func ToPtr[T any](value T) *T {
	return &value
}

// nolint: revive
func LogAndTerminateContainers(
	ctx context.Context,
	tb testing.TB,
	mockManagementPlaneContainer testcontainers.Container,
	agentContainer testcontainers.Container,
	expectNoErrorsInLogs bool,
) {
	tb.Helper()

	tb.Log("Logging nginx agent container logs")
	logReader, err := agentContainer.Logs(ctx)
	require.NoError(tb, err)

	buf, err := io.ReadAll(logReader)
	require.NoError(tb, err)
	logs := string(buf)

	tb.Log(logs)
	if expectNoErrorsInLogs {
		assert.NotContains(tb, logs, "level=ERROR", "agent log file contains logs at error level")
	}

	err = agentContainer.Terminate(ctx)
	require.NoError(tb, err)

	if mockManagementPlaneContainer != nil {
		tb.Log("Logging mock management container logs")
		logReader, err = mockManagementPlaneContainer.Logs(ctx)
		require.NoError(tb, err)

		buf, err = io.ReadAll(logReader)
		require.NoError(tb, err)
		logs = string(buf)

		tb.Log(logs)

		err = mockManagementPlaneContainer.Terminate(ctx)
		require.NoError(tb, err)
	}
}

func Env(tb testing.TB, envKey string) string {
	tb.Helper()

	envValue := os.Getenv(envKey)
	tb.Logf("Environment variable %s is set to %s", envKey, envValue)

	require.NotEmptyf(tb, envValue, "Environment variable %s should not be empty", envKey)

	return envValue
}

func ExecuteCommand(container testcontainers.Container, cmd []string) (string, error) {
	exitCode, response, err := container.Exec(context.Background(), cmd, tcexec.Multiplexed())
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

func TestAgentHasNoErrorLogs(t *testing.T, agentContainer testcontainers.Container) {
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

// CreateContainerNetwork creates and configures a container network.
func CreateContainerNetwork(ctx context.Context, tb testing.TB) *testcontainers.DockerNetwork {
	tb.Helper()
	containerNetwork, err := network.New(ctx, network.WithAttachable())
	require.NoError(tb, err)
	tb.Cleanup(func() {
		networkErr := containerNetwork.Remove(ctx)
		tb.Logf("Error removing container network: %v", networkErr)
	})

	return containerNetwork
}
