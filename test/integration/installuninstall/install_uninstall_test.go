// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package installuninstall

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	maxFileSize                 = int64(30000000)
	maxInstallTime              = 30 * time.Second
	absContainerAgentPackageDir = "../agent/build"
	agentPackageName            = "nginx-agent"
)

var (
	osRelease   = os.Getenv("OS_RELEASE")
	packageName = os.Getenv("PACKAGE_NAME")

	expectedInstallLogMsgs = map[string]string{
		"InstallFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentSuccess":    "NGINX Agent package has been successfully installed.",
		"InstallAgentStartCmd":   "sudo systemctl start nginx-agent",
	}

	agentConfigFile = "/etc/nginx-agent/nginx-agent.conf"
)

func installUninstallSetup(tb testing.TB, expectNoErrorsInLogs bool) (testcontainers.Container, func(tb testing.TB)) {
	tb.Helper()
	ctx := context.Background()

	params := &helpers.Parameters{
		NginxConfigPath: "../../config/nginx/nginx.conf",
		LogMessage:      "nginx_pid",
	}

	// start container without agent installed
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

func TestInstallUninstall(t *testing.T) {
	testContainer, teardownTest := installUninstallSetup(t, true)
	defer teardownTest(t)
	ctx := context.Background()

	// verify size of agent package and get path to agent package
	containerAgentPackagePath := verifyAgentPackage(t, testContainer)

	// check agent is installed successfully
	verifyAgentInstall(ctx, t, testContainer, containerAgentPackagePath)

	// check output of nginx-agent --version command
	verifyAgentVersion(ctx, t, testContainer)

	// verify agent is uninstalled successfully
	verifyAgentUninstall(ctx, t, testContainer)
}

func verifyAgentPackage(tb testing.TB, testContainer testcontainers.Container) string {
	tb.Helper()
	agentPkgPath, filePathErr := filepath.Abs("../../../build/")
	require.NoError(tb, filePathErr, "Error finding local agent package build dir")

	localAgentPkg, packageErr := os.Stat(packagePath(agentPkgPath, osRelease))
	require.NoError(tb, packageErr, "Error accessing package at: "+agentPkgPath)

	// Check the file size is less than or equal 30MB
	assert.LessOrEqual(tb, localAgentPkg.Size(), maxFileSize)

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		updateDebRepo(tb, testContainer)
	}

	return packagePath(absContainerAgentPackageDir, osRelease)
}

func verifyAgentInstall(ctx context.Context, tb testing.TB, testContainer testcontainers.Container,
	containerAgentPackagePath string,
) {
	tb.Helper()

	installLog, installTime := installAgent(ctx, tb, testContainer, osRelease, containerAgentPackagePath)

	assert.LessOrEqual(tb, installTime, maxInstallTime)

	if nginxIsRunning(ctx, testContainer) {
		expectedInstallLogMsgs["InstallAgentToRunAs"] = "nginx-agent will be configured to run as same user"
	}

	for _, logMsg := range expectedInstallLogMsgs {
		assert.Contains(tb, installLog, logMsg)
	}

	_, copyFileErr := testContainer.CopyFileFromContainer(ctx, agentConfigFile)
	require.NoError(tb, copyFileErr, "filePath", agentConfigFile)
}

func verifyAgentUninstall(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) {
	tb.Helper()
	uninstallLog := uninstallAgent(ctx, tb, testContainer, osRelease)

	expectedUninstallLogMsgs := make(map[string]string)
	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		expectedUninstallLogMsgs["UninstallAgent"] = "Removing nginx-agent"
		expectedUninstallLogMsgs["UninstallAgentPurgingFiles"] = "Purging configuration files for nginx-agent"
	} else if strings.Contains(osRelease, "alpine") {
		expectedUninstallLogMsgs["UninstallAgent"] = "Purging nginx-agent"
	} else {
		expectedUninstallLogMsgs["UninstallAgent"] = "Removed:\n  nginx-agent"
	}

	for _, logMsg := range expectedUninstallLogMsgs {
		assert.Contains(tb, uninstallLog, logMsg)
	}

	_, copyFileErr := testContainer.CopyFileFromContainer(ctx, agentConfigFile)
	require.Error(tb, copyFileErr, "filePath", agentConfigFile)
}

func verifyAgentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) {
	tb.Helper()

	replacer := strings.NewReplacer("nginx-agent-", "v", "SNAPSHOT-", "")
	packageVersion := replacer.Replace(os.Getenv("PACKAGE_NAME"))
	expectedVersionOutput := "nginx-agent version " + packageVersion

	exitCode, cmdOut, err := testContainer.Exec(ctx, []string{"nginx-agent", "--version"})
	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)

	stdoutStderr, readAllErr := io.ReadAll(cmdOut)
	versionOutput := helpers.RemoveASCIIControlSignals(tb, string(stdoutStderr))
	require.NoError(tb, readAllErr)
	assert.Equal(tb, expectedVersionOutput, versionOutput)
}

func installAgent(ctx context.Context, tb testing.TB, container testcontainers.Container, osReleaseContent,
	agentPackageFilePath string,
) (string, time.Duration) {
	tb.Helper()
	installCmd := createInstallCommand(osReleaseContent, agentPackageFilePath)

	start := time.Now()

	exitCode, cmdOut, err := container.Exec(ctx, installCmd)
	require.NoError(tb, err)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	msg := fmt.Sprintf("expected error code of 0 from cmd %q. Got: %v\n %s", installCmd, exitCode, stdoutStderr)
	assert.Equal(tb, 0, exitCode, msg)

	return string(stdoutStderr), time.Since(start)
}

func uninstallAgent(ctx context.Context, tb testing.TB, container testcontainers.Container,
	osReleaseContent string,
) string {
	tb.Helper()
	uninstallCmd := createUninstallCommand(osReleaseContent)

	exitCode, cmdOut, err := container.Exec(ctx, uninstallCmd)
	require.NoError(tb, err)

	msg := fmt.Sprintf("expected error code of 0 from cmd %q. Got: %v", uninstallCmd, exitCode)
	assert.Equal(tb, 0, exitCode, msg)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	return string(stdoutStderr)
}

func updateDebRepo(tb testing.TB, testContainer testcontainers.Container) {
	tb.Helper()
	updateCmd := []string{"apt-get", "update"}

	exitCode, _, err := testContainer.Exec(context.Background(), updateCmd)
	require.NoError(tb, err)

	msg := fmt.Sprintf("expected error code of 0 from cmd %q", exitCode)
	assert.Equal(tb, 0, exitCode, msg)
}

func nginxIsRunning(ctx context.Context, container testcontainers.Container) bool {
	exitCode, _, err := container.Exec(ctx, []string{"pgrep", "nginx"})
	if err != nil || exitCode != 0 {
		return false
	}

	return true
}

func createUninstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "ubuntu") || strings.Contains(osReleaseContent, "debian") {
		return []string{"apt", "purge", "-y", agentPackageName}
	} else if strings.Contains(osReleaseContent, "alpine") {
		return []string{"apk", "del", agentPackageName}
	}

	return []string{"yum", "remove", "-y", agentPackageName}
}

func createInstallCommand(osReleaseContent, agentPackageFilePath string) []string {
	if strings.Contains(osReleaseContent, "ubuntu") || strings.Contains(osReleaseContent, "debian") {
		return []string{"dpkg", "-i", agentPackageFilePath}
	} else if strings.Contains(osReleaseContent, "alpine") {
		return []string{"apk", "add", "--allow-untrusted", agentPackageFilePath}
	}

	return []string{"yum", "localinstall", "-y", agentPackageFilePath}
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
