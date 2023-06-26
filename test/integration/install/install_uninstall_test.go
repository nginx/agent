package install

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

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
	AGENT_PACKAGE_REPO = os.Getenv("PACKAGES_REPO")
)

// TestAgentManualInstallUninstall tests Agent Install and Uninstall.
// Verifies that agent installs with correct output and files.
// Verifies that agent uninstalls and removes all the files.
func TestAgentManualInstallUninstall(t *testing.T) {
	expectedInstallLogMsgs := map[string]string{
		"InstallFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentSuccess":    "NGINX Agent package has been successfully installed.",
		"InstallAgentStartCmd":   "sudo systemctl start nginx-agent",
	}

	expectedAgentPaths := map[string]string{
		"AgentConfigFile":        "/etc/nginx-agent/nginx-agent.conf",
		"AgentDynamicConfigFile": "/var/lib/nginx-agent/agent-dynamic.conf",
	}

	require.NotEmpty(t, AGENT_PACKAGE_REPO, "Environment variable $PACKAGE_REPO not set")

	testContainer := utils.SetupTestContainerWithoutAgent(t)

	ctx := context.Background()

	osReleaseContent, err := getOsReleaseContent(ctx, testContainer)
	require.NoError(t, err)

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		err := updateRepo(testContainer, osReleaseContent)
		require.NoError(t, err, "failed to update repo packages cache")
	}

	installLog, installTime, err := installAgent(ctx, testContainer, osReleaseContent)
	require.NoError(t, err)

	assert.LessOrEqual(t, installTime, maxInstallTime)

	if nginxIsRunning(ctx, testContainer) {
		expectedInstallLogMsgs["InstallAgentToRunAs"] = "nginx-agent will be configured to run as same user"
	}

	for _, logMsg := range expectedInstallLogMsgs {
		assert.Contains(t, installLog, logMsg)
	}

	// Check nginx-agent config files were created.
	for _, path := range expectedAgentPaths {
		_, err = testContainer.CopyFileFromContainer(ctx, path)
		assert.NoError(t, err)
	}

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
	for path := range expectedAgentPaths {
		_, err = testContainer.CopyFileFromContainer(ctx, path)
		assert.Error(t, err)
	}
}

// installAgent installs the agent returning total install time and install output
func installAgent(ctx context.Context, container *testcontainers.DockerContainer, osReleaseContent string) (string, time.Duration, error) {
	start := time.Now()

	installCmd := createInstallCommand(osReleaseContent)

	exitCode, cmdOut, err := container.Exec(ctx, installCmd)
	if err != nil {
		return "", time.Since(start), fmt.Errorf("failed to install agent: %v", err)
	}
	stdoutStderr, _ := io.ReadAll(cmdOut)

	if exitCode != 0 {
		return "", time.Since(start), fmt.Errorf("expected error code of 0. Got: %v\n %s", exitCode, stdoutStderr)
	}

	return string(stdoutStderr), time.Since(start), err
}

// uninstallAgent uninstall the agent returning output
func uninstallAgent(ctx context.Context, container *testcontainers.DockerContainer, osReleaseContent string) (string, error) {
	uninstallCmd := createUninstallCommand(osReleaseContent)

	exitCode, cmdOut, err := container.Exec(ctx, uninstallCmd)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("expected error code of 0. Got: %v", exitCode)
	}

	stdoutStderr, err := io.ReadAll(cmdOut)
	return string(stdoutStderr), err
}

func updateRepo(testContainer *testcontainers.DockerContainer, osReleaseContent string) error {
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

func createInstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"apt-get", "install", "-y", agentPackageName}
	} else if strings.Contains(osReleaseContent, "alpine") {
		return []string{"apk", "add", "nginx-agent@nginx-agent"}
	} else {
		return []string{"yum", "install", "-y", agentPackageName}
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

func nginxIsRunning(ctx context.Context, container *testcontainers.DockerContainer) bool {
	exitCode, _, err := container.Exec(ctx, []string{"pgrep", "nginx"})
	if err != nil || exitCode != 0 {
		return false
	}
	return true
}

func getOsReleaseContent(ctx context.Context, testContainer *testcontainers.DockerContainer) (string, error) {
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
