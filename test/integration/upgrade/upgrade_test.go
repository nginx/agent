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
	maxFileSize    int64 = 70000000
	maxUpgradeTime       = 30 * time.Second
	agentBuildDir        = "../agent/build"
)

var (
	osRelease   = os.Getenv("OS_RELEASE")
	packageName = os.Getenv("PACKAGE_NAME")
)

func Test_UpgradeFromV3(t *testing.T) {
	ctx := context.Background()

	containerNetwork := utils.CreateContainerNetwork(ctx, t)
	utils.SetupMockManagementPlaneGrpc(ctx, t, containerNetwork)
	defer func(ctx context.Context) {
		err := utils.MockManagementPlaneGrpcContainer.Terminate(ctx)
		require.NoError(t, err)
	}(ctx)

	testContainer, teardownTest := upgradeSetup(t, true, containerNetwork)
	defer teardownTest(t)

	slog.Info("starting agent v3 upgrade tests")

	// get currently installed agent version
	oldVersion := agentVersion(ctx, t, testContainer)

	// verify agent upgrade
	verifyAgentUpgrade(ctx, t, testContainer)

	// verify version of agent
	verifyAgentVersion(ctx, t, testContainer, oldVersion)

	// Verify Agent Package Path & get the path
	verifyAgentPackageSize(t)

	// verify agent v3 config has not changed
	validateAgentConfig(ctx, t, testContainer)

	// validate agent manifest file
	verifyManifestFile(ctx, t, testContainer)

	slog.Info("finished agent v3 upgrade tests")
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

	// cmdOut for validating logs
	_, upgradeTime := upgradeAgent(ctx, tb, testContainer)

	assert.LessOrEqual(tb, upgradeTime, maxUpgradeTime)
	tb.Log("Upgrade time: ", upgradeTime)
}

func upgradeAgent(ctx context.Context, tb testing.TB, testContainer testcontainers.Container,
) (io.Reader, time.Duration) {
	tb.Helper()

	var upgradeCmd []string

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		upgradeCmd = []string{
			"apt-get", "install", "-y", "--only-upgrade",
			"/agent/build/" + packageName + ".deb", "-o", "Dpkg::Options::=--force-confold",
		}
	} else if strings.Contains(osRelease, "alpine") {
		upgradeCmd = []string{
			"apk", "add", "--allow-untrusted", "/agent/build/" + packageName + ".apk",
		}
	} else {
		upgradeCmd = []string{"yum", "reinstall", "-y", "/agent/build/" + packageName + ".rpm"}
	}

	start := time.Now()

	exitCode, cmdOut, err := testContainer.Exec(ctx, upgradeCmd)
	require.NoError(tb, err)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	output := strings.TrimSpace(string(stdoutStderr))

	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)
	tb.Logf("Upgrade command output: %s", output)

	duration := time.Since(start)

	return cmdOut, duration
}

func verifyAgentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container, oldVersion string) {
	tb.Helper()

	newVersion := agentVersion(ctx, tb, testContainer)
	assert.NotEqual(tb, oldVersion, newVersion)
	tb.Logf("agent upgraded to version %s successfully", newVersion)
}

func agentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) string {
	tb.Helper()

	cmd := []string{"nginx-agent", "--version"}
	exitCode, cmdOut, err := testContainer.Exec(ctx, cmd)
	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	output := strings.TrimSpace(string(stdoutStderr))

	return output
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

func validateAgentConfig(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) {
	tb.Helper()

	agentConfigContent, err := testContainer.CopyFileFromContainer(ctx, "/etc/nginx-agent/nginx-agent.conf")
	require.NoError(tb, err)

	agentConfig, err := io.ReadAll(agentConfigContent)
	require.NoError(tb, err)

	expectedConfig, err := os.ReadFile("./configs/nginx-agent-v3-valid-config.conf")
	require.NoError(tb, err)

	expectedConfig = bytes.TrimSpace(expectedConfig)
	agentConfig = bytes.TrimSpace(agentConfig)

	assert.Equal(tb, string(expectedConfig), string(agentConfig))
	tb.Log("agent config:", string(agentConfig))
}

func verifyManifestFile(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) {
	tb.Helper()

	var manifestFileContent io.ReadCloser
	var err error

	retries := 5
	for i := range retries {
		manifestFileContent, err = testContainer.CopyFileFromContainer(ctx, "/var/lib/nginx-agent/manifest.json")
		if err == nil {
			break
		}
		tb.Logf("Error copying manifest file, retry %d/%d: %v", i+1, retries, err)
		time.Sleep(2 * time.Second)
	}

	require.NoError(tb, err)

	manifestFile, err := io.ReadAll(manifestFileContent)
	require.NoError(tb, err)

	expected := `{
  "/etc/nginx/nginx.conf": {
    "manifest_file_meta": {
      "name": "/etc/nginx/nginx.conf",
      "hash": "XEaOA4w+aT5fmNMISPwavBroLVYlkJf9sjKFTnWkTP8=",
      "size": 1142,
      "referenced": true
    }
  }
}`

	assert.Equal(tb, expected, string(manifestFile))
}
