package install

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
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

	osReleasePath               = "/etc/os-release"
	absContainerAgentPackageDir = "/agent/build"
)

var (
	AGENT_PACKAGE_FILENAME = os.Getenv("PACKAGE_NAME")
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

	expectedUninstallLogMsgs := map[string]string{
		"UninstallAgent":             "Removing nginx-agent",
		"UninstallAgentPurgingFiles": "Purging configuration files for nginx-agent",
	}

	expectedAgentPaths := map[string]string{
		"AgentConfigFile":        "/etc/nginx-agent/nginx-agent.conf",
		"AgentDynamicConfigFile": "/etc/nginx-agent/agent-dynamic.conf",
	}

	// Check the environment variable $PACKAGE_NAME is set
	require.NotEmpty(t, AGENT_PACKAGE_FILENAME, "Environment variable $PACKAGE_NAME not set")

	testContainer := utils.SetupTestContainerWithoutAgent(t)

	ctx := context.Background()
	exitCode, osReleaseFileContent, err := testContainer.Exec(ctx, []string{"cat", osReleasePath})
	assert.NoError(t, err)
	osReleaseContent, err := io.ReadAll(osReleaseFileContent)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.NotEmpty(t, osReleaseContent, "os release file empty")

	// Check the file size is less than or equal 20MB
	absLocalAgentPkgDirPath, err := filepath.Abs("../../../build/")
	assert.NoError(t, err, "Error finding local agent package build dir")
	localAgentPkg, err := os.Stat(getPackagePath(absLocalAgentPkgDirPath, string(osReleaseContent)))
	assert.NoError(t, err, "Error accessing package at: "+absLocalAgentPkgDirPath)

	assert.LessOrEqual(t, localAgentPkg.Size(), maxFileSize)

	// Install Agent inside container and record installation time/install output
	containerAgentPackagePath := getPackagePath(absContainerAgentPackageDir, string(osReleaseContent))
	installTime, installLog, err := installAgent(ctx, testContainer, containerAgentPackagePath, string(osReleaseContent))
	require.NoError(t, err)

	// Check the install time under 30s
	assert.LessOrEqual(t, installTime, maxInstallTime)

	// Check install output
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

	// Uninstall the agent package
	uninstallLog, err := uninstallAgent(ctx, testContainer, string(osReleaseContent))
	require.NoError(t, err)

	// Check uninstall output
	if strings.HasSuffix(containerAgentPackagePath, "rpm") {
		expectedUninstallLogMsgs["UninstallAgent"] = "Removed:\n  nginx-agent"
		delete(expectedUninstallLogMsgs, "UninstallAgentPurgingFiles")
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
func installAgent(ctx context.Context, container *testcontainers.DockerContainer, agentPackageFilePath, osReleaseContent string) (time.Duration, string, error) {
	// Get OS to create install cmd
	installCmd := createInstallCommand(agentPackageFilePath, osReleaseContent)

	// Start install timer
	start := time.Now()

	// Start agent installation and capture install output
	exitCode, cmdOut, err := container.Exec(ctx, installCmd)
	if err != nil {
		return time.Since(start), "", err
	}
	if exitCode != 0 {
		return time.Since(start), "", errors.New("expected exit code of 0")
	}

	stdoutStderr, err := io.ReadAll(cmdOut)
	return time.Since(start), string(stdoutStderr), err
}

// uninstallAgent uninstall the agent returning output
func uninstallAgent(ctx context.Context, container *testcontainers.DockerContainer, osReleaseContent string) (string, error) {
	// Get OS to create uninstall cmd
	uninstallCmd := createUninstallCommand(osReleaseContent)

	// Start agent uninstall and capture uninstall output
	exitCode, cmdOut, err := container.Exec(ctx, uninstallCmd)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", errors.New("expected exit code of 0")
	}

	stdoutStderr, err := io.ReadAll(cmdOut)
	return string(stdoutStderr), err
}

func createInstallCommand(agentPackageFilePath, osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"dpkg", "-i", agentPackageFilePath}
	} else {
		return []string{"yum", "localinstall", "-y", agentPackageFilePath}
	}
}

func createUninstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"apt", "purge", "-y", "nginx-agent"}
	} else {
		return []string{"yum", "remove", "-y", "nginx-agent"}
	}
}

func nginxIsRunning(ctx context.Context, container *testcontainers.DockerContainer) bool {
	exitCode, _, err := container.Exec(ctx, []string{"pgrep", "nginx"})
	if err != nil || exitCode != 0 {
		return false
	}
	return true
}

func getPackagePath(pkgDir, osReleaseContent string) string {
	pkgPath := path.Join(pkgDir, AGENT_PACKAGE_FILENAME)

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return pkgPath + ".deb"
	} else if strings.Contains(osReleaseContent, "rhel") || strings.Contains(osReleaseContent, "centos") {
		return pkgPath + ".rpm"
	} else {
		return pkgPath + ".apk"
	}
}
