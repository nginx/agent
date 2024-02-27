// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const configFilePermissions = 0o700

//nolint:ireturn
func StartContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	waitForLog, nginxConfigPath string,
) testcontainers.Container {
	tb.Helper()

	containerOSType := getEnv(tb, "CONTAINER_OS_TYPE")
	packageName := getEnv(tb, "PACKAGE_NAME")
	packageRepo := getEnv(tb, "PACKAGES_REPO")
	baseImage := getEnv(tb, "BASE_IMAGE")
	osRelease := getEnv(tb, "OS_RELEASE")
	osVersion := getEnv(tb, "OS_VERSION")
	buildTarget := getEnv(tb, "BUILD_TARGET")

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../",
			Dockerfile:    fmt.Sprintf("./scripts/docker/nginx-oss/%s/Dockerfile", containerOSType),
			KeepImage:     false,
			PrintBuildLog: true,
			BuildArgs: map[string]*string{
				"PACKAGE_NAME":  ToPtr(packageName),
				"PACKAGES_REPO": ToPtr(packageRepo),
				"BASE_IMAGE":    ToPtr(baseImage),
				"OS_RELEASE":    ToPtr(osRelease),
				"OS_VERSION":    ToPtr(osVersion),
				"ENTRY_POINT":   ToPtr("./scripts/docker/entrypoint.sh"),
			},
			BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
				buildOptions.Target = buildTarget
			},
		},
		ExposedPorts: []string{"9091/tcp"},
		WaitingFor:   wait.ForLog(waitForLog),
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
				HostFilePath:      "./nginx-agent.conf",
				ContainerFilePath: "/etc/nginx-agent/nginx-agent.conf",
				FileMode:          configFilePermissions,
			},
			{
				HostFilePath:      nginxConfigPath,
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

//nolint:ireturn
func StartMockManagementPlaneContainer(
	ctx context.Context,
	tb testing.TB,
	containerNetwork *testcontainers.DockerNetwork,
	nginxConfigPath string,
) testcontainers.Container {
	tb.Helper()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../",
			Dockerfile:    "./test/docker/mock-management-plane/Dockerfile",
			KeepImage:     false,
			PrintBuildLog: true,
		},
		ExposedPorts: []string{"9092/tcp"},
		Networks: []string{
			containerNetwork.Name,
		},
		NetworkAliases: map[string][]string{
			containerNetwork.Name: {
				"managementPlane",
			},
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      nginxConfigPath,
				ContainerFilePath: "/mock-management-plane/config/etc/nginx/nginx.conf",
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

func getEnv(tb testing.TB, envKey string) string {
	tb.Helper()

	envValue := os.Getenv(envKey)
	tb.Logf("Environment variable %s is set to %s", envKey, envValue)
	require.NotEmptyf(tb, envValue, "Environment variable %s should not be empty", envKey)

	return envValue
}
