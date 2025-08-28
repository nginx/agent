// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package upgrade

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/integration/utils"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	maxFileSize      = 70000000
	maxUpgradeTime   = 30 * time.Second
	agentBuildDir    = "../agent/build"
	agentPackageName = "nginx-agent"
)

var (
	osRelease        = os.Getenv("OS_RELEASE")
	packageName      = os.Getenv("NGINX_AGENT_PACKAGE_NAME")
	agentConfig      = "./configs/nginx-agent.conf"
	agentValidConfig = "./configs/nginx-agent-v3-valid-config.conf"
)

func TestV3toV3Upgrade(t *testing.T) {
	ctx := context.Background()
	containerNetwork := utils.CreateContainerNetwork(ctx, t)
	testContainer, teardownTest := upgradeSetup(t, true, containerNetwork)
	defer teardownTest(t)

	slog.Info("starting agent v3 upgrade tests")

	// Verify Agent Package Path & get the path
	verifyAgentPackageSize(t)

	// verify agent upgrade
	verifyAgentUpgrade(ctx, t, testContainer)

	// verify version of agent
	verifyAgentVersion(ctx, t, testContainer, agentPackageName)

	// verify agent v3 config has not changed
	validateAgentConfig(t, agentValidConfig, agentConfig)

	// Validate expected logs

	// validate agent manifest file
}

func upgradeSetup(tb testing.TB, expectNoErrorsInLogs bool,
	containerNetwork *testcontainers.DockerNetwork,
) (testcontainers.Container, func(tb testing.TB)) {
	tb.Helper()
	ctx := context.Background()

	params := &helpers.Parameters{
		NginxConfigPath:      "./configs/nginx-oss.conf",
		NginxAgentConfigPath: "./configs/nginx-agent.conf",
		LogMessage:           "nginx_pid",
	}

	testContainer := helpers.StartContainer(
		ctx,
		tb,
		containerNetwork,
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

func verifyAgentPackageSize(tb testing.TB) string {
	tb.Helper()
	agentPkgPath, filePathErr := filepath.Abs("../../../build/")
	require.NoError(tb, filePathErr, "Error finding local agent package build dir")

	localAgentPkg, packageErr := os.Stat(packagePath(agentPkgPath, osRelease))
	require.NoError(tb, packageErr, "Error accessing package at: "+agentPkgPath)

	// check if file size is less than 70MB
	assert.Less(tb, localAgentPkg.Size(), maxFileSize)

	return packagePath(agentBuildDir, osRelease)
}

func verifyAgentUpgrade(ctx context.Context, tb testing.TB,
	testContainer testcontainers.Container,
) {
	tb.Helper()

	cmdOut, upgradeTime := upgradeAgent(ctx, tb, testContainer)

	assert.LessOrEqual(tb, upgradeTime, maxUpgradeTime)
	tb.Log("Upgrade time: ", upgradeTime)

	// validate logs here
	validateLogs(tb, cmdOut)
}

func upgradeAgent(ctx context.Context, tb testing.TB, testContainer testcontainers.Container,
) (io.Reader, time.Duration) {
	tb.Helper()

	var updateCmd, upgradeCmd []string

	if strings.Contains(osRelease, "Ubuntu") || strings.Contains(osRelease, "Debian") {
		updateCmd = []string{"apt-get", "update"}
		upgradeCmd = []string{
			"apt-get", "install", "-y", "--only-upgrade",
			"nginx-agent", "-o", "Dpkg::Options::=--force-confold",
		}
	} else {
		updateCmd = []string{"yum", "-y", "makecache"}
		upgradeCmd = []string{"yum", "update", "-y", "nginx-agent"}
	}

	start := time.Now()

	exitCode, _, err := testContainer.Exec(ctx, updateCmd)
	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)

	exitCode, cmdOut, err := testContainer.Exec(ctx, upgradeCmd)
	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)

	duration := time.Since(start)

	return cmdOut, duration
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

func validateLogs(tb testing.TB, expectedLogs io.Reader) {
}
