// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package utils

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
func StartContainer(ctx context.Context, tb testing.TB, waitForLog string) testcontainers.Container {
	tb.Helper()
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:       "../../",
			Dockerfile:    fmt.Sprintf("./scripts/docker/nginx-oss/%s/Dockerfile", os.Getenv("CONTAINER_OS_TYPE")),
			KeepImage:     false,
			PrintBuildLog: true,
			BuildArgs: map[string]*string{
				"PACKAGE_NAME":  toPtr(os.Getenv("PACKAGE_NAME")),
				"PACKAGES_REPO": toPtr(os.Getenv("PACKAGES_REPO")),
				"BASE_IMAGE":    toPtr(os.Getenv("BASE_IMAGE")),
				"OS_RELEASE":    toPtr(os.Getenv("OS_RELEASE")),
				"OS_VERSION":    toPtr(os.Getenv("OS_VERSION")),
				"ENTRY_POINT":   toPtr("./scripts/docker/entrypoint.sh"),
			},
			BuildOptionsModifier: func(buildOptions *types.ImageBuildOptions) {
				buildOptions.Target = os.Getenv("BUILD_TARGET")
			},
		},
		ExposedPorts: []string{"9091/tcp"},
		WaitingFor:   wait.ForLog(waitForLog),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "./nginx-agent.conf",
				ContainerFilePath: "/etc/nginx-agent/nginx-agent.conf",
				FileMode:          configFilePermissions,
			},
			{
				HostFilePath:      "./nginx.conf",
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

func toPtr[T any](value T) *T {
	return &value
}
