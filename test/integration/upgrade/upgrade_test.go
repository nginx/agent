// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package upgrade

import (
	"bytes"
	"context"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	maxFileSize      = 70000000
	maxUpgradeTime   = 30 * time.Second
	agentBuildDir    = "../agent/build"
	agentPackageName = "nginx-agent"
)

var (
	osRelease      = os.Getenv("OS_RELEASE")
	oldPackageName = os.Getenv("OLD_PACKAGE_NAME")
	packageName    = os.Getenv("PACKAGE_NAME")
)

func TestV3toV3Upgrade(t *testing.T) {
	ctx := context.Background()
	testContainer, teardownTest := upgradeSetup(t, true)
	defer teardownTest(t)

	slog.Info("starting upgrade to latest agent v3 tests")

	// Verify Agent Package Path & Install Agent
	agentPackagePath := verifyAgentPackageSize(t, testContainer)

	// verify agent upgrade
	upgradeAgent(ctx, t, testContainer)

	// verify version of agent
	verifyAgentVersion(ctx, t, testContainer, agentPackageName)

	// verify agent v3 config has not changed

	// Validate expected logs

	// validate agent manifest file

	// verify agent package size and get its path
	packagePath := verifyAgentPackage(t, testContainer)

	// verify size of agent package and get path to agent package
	containerAgentPackagePath := verifyAgentPackage(t, testContainer)
}

func upgradeSetup(tb testing.TB, expectNoErrorsInLogs bool) (testcontainers.Container, func(tb testing.TB)) {
	tb.Helper()
	ctx := context.Background()

	params := &helpers.Parameters{
		NginxConfigPath: "./config/nginx/nginx.conf",
		LogMessage:      "nginx_pid",
	}

	testContainer := helpers.StartAgentlessContainer(
		ctx,
		tb,
		params,
	)

	return testContainer, func(tb testing.TB) {
		tb.Helper()
		helpers.LogAndTerminateContainers(
			ctx,
			tb,
			nil,
			testContainer,
			expectNoErrorsInLogs,
			nil,
		)
	}
}

func upgradeAgent(tb testing.TB, testContainer testcontainers.Container) (string, time.Duration) {
	tb.Helper()

	var updateCmd, upgradeCmd []string

	if strings.Contains(osRelease, "Ubuntu") || strings.Contains(osRelease, "Debian") {
		updateCmd = []string{"apt-get", "update"}
		upgradeCmd = []string{"apt-get", "install", "-y", "--only-upgrade", "nginx-agent", "-o", "Dpkg::Options::=--force-confold"}
	} else {
		updateCmd = []string{"yum", "-y", "makecache"}
		upgradeCmd = []string{"yum", "update", "-y", "nginx-agent"}
	}

	start := time.Now()

	var output []byte

	exitCode, cmdOut, err := testContainer.Exec(ctx, updateCmd)
	require.NoError(tb, err)
	stdourStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)
	output = append(output, stdourStderr...)
	assert.Equal(tb, 0, exitCode)

	exitCode, cmdOut, err = testContainer.Exec(ctx, upgradeCmd)
	require.NoError(tb, err)
	stdourStderr, err = io.ReadAll(cmdOut)
	require.NoError(tb, err)
	output = append(output, stdourStderr...)
	assert.Equal(tb, 0, exitCode)

	duration := time.Since(start)
	return string(output), duration

}

func verifyAgentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container, oldVersion string) {
	tb.Helper()
	cmd := []string{"nginx-agent", "--version"}
	exitCode, cmdOut, err := testContainer.Exec(ctx, cmd)
	require.NoError(tb, err)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	output := strings.TrimSpace(string(stdoutStderr))

	assert.Equal(tb, 0, exitCode)
	if output != oldVersion {
		tb.Logf("expected version %s, got %s", oldVersion, output)
	}
	tb.Logf("agent upgraded to version %s successfully", output)
}

func verifyAgentPackageSize(tb testing.TB, testContainer testcontainers.Container) string {
	tb.Helper()
	agentPkgPath, filePathErr := filepath.Abs("../../../build/")
	require.NoError(tb, filePathErr, "Error finding local agent package build dir")

	localAgentPkg, packageErr := os.Stat(packagePath(agentPkgPath, osRelease))
	require.NoError(tb, packageErr, "Error accessing package at: "+agentPkgPath)

	// Check if file size is less than 70MB
	assert.Less(tb, localAgentPkg.Size(), maxFileSize)

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		upgradeAgent(tb, testContainer)
	}

	return packagePath(agentBuildDir, osRelease)
}

func packagePath(pkgDir, osReleaseContent string) string {
	pkgPath := path.Join(pkgDir, packageName)

	if strings.Contains(osReleaseContent, "ubuntu") || strings.Contains(osReleaseContent, "Debian") {
		return pkgPath + ".deb"
	} else if strings.Contains(osReleaseContent, "alpine") {
		return pkgPath + ".apk"
	}

	return pkgPath + ".rpm"
}

func validateAgentConfig(tb testing.TB, expectedConfigPath, updatedConfigPath string) {
	tb.Helper()

	// valid config file
	expectedContent, err := os.ReadFile(expectedConfigPath)
	require.NoError(tb, err)

	// new config file
	updated, err := os.ReadFile(updatedConfigPath)
	require.NoError(tb, err)

	if !bytes.Equal(expectedContent, updated) {
		tb.Fatalf("expected no changes in the config file")
	}
	tb.Logf("config file validation was successful")
}
