package install

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

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/test/integration/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	maxFileSize    = int64(20000000)
	maxInstallTime = 30 * time.Second

	agentPackageName            = "nginx-agent"
	osReleasePath               = "/etc/os-release"
	absContainerAgentPackageDir = "/agent/build"
)

var (
	AGENT_PACKAGE_FILENAME = os.Getenv("PACKAGE_NAME")
	INSTALL_FROM_REPO      = os.Getenv("INSTALL_FROM_REPO")
)

func installUninstallSetup(tb testing.TB, expectNoErrorsInLogs bool) (testcontainers.Container, func(tb testing.TB)) {
	tb.Helper()
	ctx := context.Background()

	params := &utils.Parameters{
		NginxConfigPath: "./nginx.conf",
		LogMessage:      "nginx_pid",
	}

	// start container without agent installed
	testContainer := utils.StartAgentlessContainer(
		ctx,
		tb,
		params,
	)

	return testContainer, func(tb testing.TB) {
		tb.Helper()
		utils.LogAndTerminateContainers(
			ctx,
			tb,
			nil,
			testContainer,
			expectNoErrorsInLogs,
		)
	}
}

// TestAgentManualInstallUninstall tests Agent Install and Uninstall.
// Verifies that agent installs with correct output and files.
// Verifies that agent uninstalls and removes all the files.
func TestAgentManualInstallUninstall(t *testing.T) {
	log.Info("testing agent install uninstall")
	expectedInstallLogMsgs := map[string]string{
		"InstallFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentSuccess":    "NGINX Agent package has been successfully installed.",
		"InstallAgentStartCmd":   "sudo systemctl start nginx-agent",
	}

	expectedAgentPaths := map[string]string{
		"AgentConfigFile":        "/etc/nginx-agent/nginx-agent.conf",
		"AgentDynamicConfigFile": "/var/lib/nginx-agent/agent-dynamic.conf",
	}

	testContainer, teardownTest := installUninstallSetup(t, true)
	defer teardownTest(t)

	ctx := context.Background()

	osReleaseContent, err := getOsReleaseContent(ctx, testContainer)
	require.NoError(t, err)

	if INSTALL_FROM_REPO == "" {
		absLocalAgentPkgDirPath, err := filepath.Abs("../../../build/")
		assert.NoError(t, err, "Error finding local agent package build dir")
		localAgentPkg, err := os.Stat(getPackagePath(absLocalAgentPkgDirPath, osReleaseContent))
		assert.NoError(t, err, "Error accessing package at: "+absLocalAgentPkgDirPath)

		// Check the file size is less than or equal 20MB
		assert.LessOrEqual(t, localAgentPkg.Size(), maxFileSize)
	}

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		err := updateDebRepo(testContainer)
		require.NoError(t, err, "failed to update deb repo package cache")
	}

	containerAgentPackagePath := getPackagePath(absContainerAgentPackageDir, osReleaseContent)
	installLog, installTime, err := installAgent(ctx, testContainer, osReleaseContent, containerAgentPackagePath)
	require.NoError(t, err)

	assert.LessOrEqual(t, installTime, maxInstallTime)

	if nginxIsRunning(ctx, testContainer) {
		expectedInstallLogMsgs["InstallAgentToRunAs"] = "nginx-agent will be configured to run as same user"
	}

	for _, logMsg := range expectedInstallLogMsgs {
		assert.Contains(t, installLog, logMsg)
	}

	// Check nginx-agent config files were created.
	for _, agentPath := range expectedAgentPaths {
		_, err = testContainer.CopyFileFromContainer(ctx, agentPath)
		assert.NoError(t, err)
	}

	replacer := strings.NewReplacer("nginx-agent-", "v", "SNAPSHOT-", "")
	packageVersion := replacer.Replace(os.Getenv("PACKAGE_NAME"))

	expectedVersionOutput := fmt.Sprintf("nginx-agent version %s", packageVersion)

	// Check agent version command output
	versionOutput, err := checkAgentVersion(ctx, testContainer)
	assert.Equal(t, expectedVersionOutput, versionOutput)

	uninstallLog, err := uninstallAgent(ctx, testContainer, osReleaseContent)
	require.NoError(t, err)

	expectedUninstallLogMsgs := map[string]string{}
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		expectedUninstallLogMsgs["UninstallAgent"] = "Removing nginx-agent"
		expectedUninstallLogMsgs["UninstallAgentPurgingFiles"] = "Purging configuration files for nginx-agent"
	} else if strings.Contains(osReleaseContent, "alpine") {
		expectedUninstallLogMsgs["UninstallAgent"] = "Purging nginx-agent"
	} else {
		expectedUninstallLogMsgs["UninstallAgent"] = "Removed:\n  nginx-agent"
	}
	for _, logMsg := range expectedUninstallLogMsgs {
		assert.Contains(t, uninstallLog, logMsg)
	}

	// Check nginx-agent config files were removed.
	for agentPath := range expectedAgentPaths {
		_, err = testContainer.CopyFileFromContainer(ctx, agentPath)
		assert.Error(t, err)
	}
	log.Info("finished testing agent install uninstall")
}

// installAgent installs the agent returning total install time and install output
func installAgent(ctx context.Context, container testcontainers.Container, osReleaseContent string, agentPackageFilePath string) (string, time.Duration, error) {
	installCmd := createInstallCommand(osReleaseContent, agentPackageFilePath)

	start := time.Now()

	exitCode, cmdOut, err := container.Exec(ctx, installCmd)
	if err != nil {
		return "", time.Since(start), fmt.Errorf("failed to install agent: %v", err)
	}
	stdoutStderr, _ := io.ReadAll(cmdOut)
	if exitCode != 0 {
		return string(stdoutStderr), time.Since(start), fmt.Errorf("expected error code of 0 from cmd %q. Got: %v\n %s", installCmd, exitCode, stdoutStderr)
	}

	return string(stdoutStderr), time.Since(start), err
}

// uninstallAgent uninstall the agent returning output
func uninstallAgent(ctx context.Context, container testcontainers.Container, osReleaseContent string) (string, error) {
	uninstallCmd := createUninstallCommand(osReleaseContent)

	exitCode, cmdOut, err := container.Exec(ctx, uninstallCmd)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("expected error code of 0 from cmd %q. Got: %v", uninstallCmd, exitCode)
	}

	stdoutStderr, err := io.ReadAll(cmdOut)
	return string(stdoutStderr), err
}

func updateDebRepo(testContainer testcontainers.Container) error {
	if INSTALL_FROM_REPO == "" {
		return nil
	}

	updateCmd := []string{"apt-get", "update"}

	exitCode, _, err := testContainer.Exec(context.Background(), updateCmd)
	if err != nil {
		return fmt.Errorf("failed to update repo: %v", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("unexpected error code updating repo. Expected 0, got: %v", exitCode)
	}
	return nil
}

func createInstallCommand(osReleaseContent string, agentPackageFilePath string) []string {
	if INSTALL_FROM_REPO == "" {
		if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
			return []string{"dpkg", "-i", agentPackageFilePath}
		} else if strings.Contains(osReleaseContent, "alpine") {
			return []string{"apk", "add", "--allow-untrusted", agentPackageFilePath}
		} else {
			return []string{"yum", "localinstall", "-y", agentPackageFilePath}
		}
	} else {
		if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
			return []string{"apt-get", "install", "-y", agentPackageName}
		} else if strings.Contains(osReleaseContent, "alpine") {
			return []string{"apk", "add", agentPackageName}
		} else {
			return []string{"yum", "install", "-y", agentPackageName}
		}
	}
}

func createUninstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"apt", "purge", "-y", agentPackageName}
	} else if strings.Contains(osReleaseContent, "alpine") {
		return []string{"apk", "del", agentPackageName}
	} else {
		return []string{"yum", "remove", "-y", agentPackageName}
	}
}

func checkAgentVersion(ctx context.Context, container testcontainers.Container) (string, error) {
	exitCode, cmdOut, err := container.Exec(ctx, []string{"nginx-agent", "--version"})
	if err != nil {
		return "", fmt.Errorf("failed to check agent version: %v", err)
	}
	stdoutStderr, _ := io.ReadAll(cmdOut)
	if exitCode != 0 {
		return "", fmt.Errorf("expected error code of 0 from cmd got: %v\n %s", exitCode, stdoutStderr)
	}

	return strings.Trim(string(stdoutStderr), "%\x00\x01\n"), nil
}

func nginxIsRunning(ctx context.Context, container testcontainers.Container) bool {
	exitCode, _, err := container.Exec(ctx, []string{"pgrep", "nginx"})

	if err != nil || exitCode != 0 {
		return false
	}
	return true
}

func getOsReleaseContent(ctx context.Context, testContainer testcontainers.Container) (string, error) {
	exitCode, osReleaseFileContent, err := testContainer.Exec(ctx, []string{"cat", osReleasePath})
	if err != nil {
		return "", fmt.Errorf("failed to read osRelease file: %v", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("unexpected error code reading osRelease file. Expected 0, got: %v", exitCode)
	}
	osReleaseBytes, err := io.ReadAll(osReleaseFileContent)
	if err != nil {
		return "", fmt.Errorf("failed to read osRelease content: %v", err)
	}

	return string(osReleaseBytes), nil
}

func getPackagePath(pkgDir, osReleaseContent string) string {
	pkgPath := path.Join(pkgDir, AGENT_PACKAGE_FILENAME)

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return pkgPath + ".deb"
	} else if strings.Contains(osReleaseContent, "alpine") {
		return pkgPath + ".apk"
	} else {
		return pkgPath + ".rpm"
	}
}
