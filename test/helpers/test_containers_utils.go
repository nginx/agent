// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const configFilePermissions = 0o700

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

func StartNginxLessContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	parameters *Parameters,
) testcontainers.Container {
	tb.Helper()

	packageName := Env(tb, "PACKAGE_NAME")
	packageRepo := Env(tb, "PACKAGES_REPO")
	baseImage := Env(tb, "BASE_IMAGE")
	buildTarget := Env(tb, "BUILD_TARGET")
	osRelease := Env(tb, "OS_RELEASE")
	osVersion := Env(tb, "OS_VERSION")
	dockerfilePath := Env(tb, "DOCKERFILE_PATH")
	tag := Env(tb, "TAG")
	imagePath := Env(tb, "IMAGE_PATH")
	containerRegistry := Env(tb, "CONTAINER_NGINX_IMAGE_REGISTRY")

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
				"ENTRY_POINT":                    ToPtr("./test/docker/nginxless-entrypoint.sh"),
				"CONTAINER_NGINX_IMAGE_REGISTRY": ToPtr(containerRegistry),
				"IMAGE_PATH":                     ToPtr(imagePath),
				"TAG":                            ToPtr(tag),
			},
			BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
				buildOptions.Target = buildTarget
			},
		},
		ExposedPorts: []string{"9094/tcp"},
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

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(tb, err)

	return container
}

func StartMockManagementPlaneGrpcContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
) testcontainers.Container {
	tb.Helper()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../../",
			Dockerfile:    "./test/mock/grpc/Dockerfile",
			KeepImage:     false,
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"9092/tcp", "9093/tcp"},
		Networks: []string{
			containerNetwork.Name,
		},
		NetworkAliases: map[string][]string{
			containerNetwork.Name: {
				"managementPlane",
			},
		},
		WaitingFor: wait.ForLog("Starting mock management plane gRPC server"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(tb, err)

	return container
}

func StartAuxiliaryMockManagementPlaneGrpcContainer(ctx context.Context, tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
) testcontainers.Container {
	tb.Helper()
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../../",
			Dockerfile:    "./test/integration/auxiliarycommandserver/Dockerfile",
			KeepImage:     false,
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"9095/tcp", "9096/tcp"},
		Networks: []string{
			containerNetwork.Name,
		},
		NetworkAliases: map[string][]string{
			containerNetwork.Name: {
				"managementPlaneAuxiliary",
			},
		},
		WaitingFor: wait.ForLog("Starting mock management plane gRPC server"),
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
	auxiliaryMockManagementPlaneContainer testcontainers.Container,
) {
	tb.Helper()

	tb.Log("======================== Logging Agent Container Logs ========================")
	logReader, err := agentContainer.Logs(ctx)
	require.NoError(tb, err)

	buf, err := io.ReadAll(logReader)
	require.NoError(tb, err)
	logs := string(buf)

	assert.NotContains(tb, logs, "manifest file is empty",
		"Error reading manifest file found in agent log")
	tb.Log(logs)
	if expectNoErrorsInLogs {
		assert.NotContains(tb, logs, "level=ERROR", "agent log file contains logs at error level")
	}

	err = agentContainer.Terminate(ctx)
	require.NoError(tb, err)

	if mockManagementPlaneContainer != nil {
		tb.Log("======================== Logging Mock Management Container Logs ========================")
		logReader, err = mockManagementPlaneContainer.Logs(ctx)
		require.NoError(tb, err)

		buf, err = io.ReadAll(logReader)
		require.NoError(tb, err)
		logs = string(buf)

		tb.Log(logs)

		err = mockManagementPlaneContainer.Terminate(ctx)
		require.NoError(tb, err)
	}

	if auxiliaryMockManagementPlaneContainer != nil {
		tb.Log("======================== Logging Auxiliary Mock Management Container Logs ========================")
		logReader, err = auxiliaryMockManagementPlaneContainer.Logs(ctx)
		require.NoError(tb, err)

		buf, err = io.ReadAll(logReader)
		require.NoError(tb, err)
		logs = string(buf)

		tb.Log(logs)

		err = auxiliaryMockManagementPlaneContainer.Terminate(ctx)
		require.NoError(tb, err)
	}
}
