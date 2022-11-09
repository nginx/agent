package install

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	osReleasePath  = "/etc/os-release"
	maxFileSize    = int64(20000000)
	maxInstallTime = 30 * time.Second
)

var (
	AGENT_PACKAGE_FILE = os.Getenv("AGENT_PACKAGE_FILE")
)

// TestAgentManualInstallUninstall tests Agent Install and Uninstall.
// Verifies that agent installs with correct output and files.
// Verifies that agent uninstalls and removes all the files.
func TestAgentManualInstallUninstall(t *testing.T) {
	expectedLogMsgs := map[string]string{
		"ConnectionStatus":           "agent_connect_response:<agent_config:<details:<> configs:<configs:<> > > status:<statusCode:CONNECT_OK > >",
		"InstallFoundNginxAgent":     "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentToRunAs":        "nginx-agent will be configured to run as same user",
		"InstallAgentSuccess":        "NGINX Agent package has been successfully installed.",
		"InstallAgentStartCmd":       "sudo systemctl start nginx-agent",
		"UninstallAgent":             "Removing nginx-agent",
		"UninstallAgentStopService":  "Stop and disable nginx-agent service",
		"UninstallAgentPurgingFiles": "Purging configuration files for nginx-agent",
	}

	expectedAgentPaths := map[string]string{
		"AgentConfigFile": "/etc/nginx-agent/nginx-agent.conf",
		"AgentSystemFile": "/etc/systemd/system/multi-user.target.wants/nginx-agent.service",
	}

	// Check the environment variable $AGENT_PACKAGE_FILE is set
	require.NotEmpty(t, AGENT_PACKAGE_FILE, "Environment variable $AGENT_PACKAGE_FILE not set")

	// Check the agent tarball is present
	file, err := os.Stat(AGENT_PACKAGE_FILE)
	require.NoError(t, err, "Error accessing tarball at: "+AGENT_PACKAGE_FILE)

	// Install Agent and record installation time/install output
	installTime, agentLog := installAgent(t, AGENT_PACKAGE_FILE)

	// Check the file size is less than or equal 20MB
	assert.LessOrEqual(t, file.Size(), maxFileSize)

	// Check the install time under 30s
	assert.LessOrEqual(t, installTime, float64(maxInstallTime))

	// Check install output
	assert.Contains(t, agentLog, expectedLogMsgs["InstallFoundNginxAgent"])
	assert.Contains(t, agentLog, expectedLogMsgs["InstallAgentToRunAs"])
	assert.Contains(t, agentLog, expectedLogMsgs["InstallAgentSuccess"])
	assert.Contains(t, agentLog, expectedLogMsgs["InstallAgentStartCmd"])

	// Check nginx-agent config is created.
	assert.FileExists(t, expectedAgentPaths["AgentConfigFile"])

	// Check nginx-agent system unit file is created.
	assert.FileExists(t, expectedAgentPaths["AgentSystemFile"])

	// Uninstall the agent package
	uninstallLog := uninstallAgent(t, "nginx-agent")

	// Check uninstall output
	assert.Contains(t, uninstallLog, expectedLogMsgs["UninstallAgent"])
	assert.Contains(t, uninstallLog, expectedLogMsgs["UninstallAgentStopService"])
	assert.Contains(t, uninstallLog, expectedLogMsgs["UninstallAgentPurgingFiles"])

	// Check nginx-agent config is removed.
	assert.NoFileExists(t, expectedAgentPaths["AgentConfigFile"])

	// Check nginx-agent system unit file is removed.
	assert.NoFileExists(t, expectedAgentPaths["AgentSystemFile"])
}

// installAgent installs the agent returning total install time and install output
func installAgent(t *testing.T, agentPackage string) (float64, string) {
	// Get OS to create install cmd
	installCmd := createInstallCommand(t)

	// Start install timer
	start := time.Now()

	// Start agent installation and capture install output
	cmd := exec.Command(installCmd[0], installCmd[1], installCmd[2], agentPackage)

	stdoutStderr, err := cmd.CombinedOutput()
	require.NoError(t, err)

	end := time.Now()
	elapsed := end.Sub(start)

	return float64(elapsed), string(stdoutStderr)
}

// uninstallAgent uninstall the agent returning output
func uninstallAgent(t *testing.T, agentPackage string) string {
	// Get OS to create uninstall cmd
	uninstallCmd := createUninstallCommand(t)

	// Start agent uninstall and capture uninstall output
	cmd := exec.Command(uninstallCmd[0], uninstallCmd[1], uninstallCmd[2], uninstallCmd[3], agentPackage)

	stdoutStderr, err := cmd.CombinedOutput()
	require.NoError(t, err)

	return string(stdoutStderr)
}

// Creates install command based on OS
func createInstallCommand(t *testing.T) []string {
	// Check OS release file exists first to determine OS
	require.FileExists(t, osReleasePath)

	content, _ := ioutil.ReadFile(osReleasePath)
	os := string(content)
	if strings.Contains(os, "UBUNTU") || strings.Contains(os, "Debian") {
		return []string{"sudo", "apt", "install"}
	} else {
		return []string{"sudo", "yum", "install"}
	}
}

// Creates uninstall command based on OS
func createUninstallCommand(t *testing.T) []string {
	// Check OS release file exists first to determine OS
	require.FileExists(t, osReleasePath)

	content, _ := ioutil.ReadFile(osReleasePath)
	os := string(content)
	if strings.Contains(os, "UBUNTU") || strings.Contains(os, "Debian") {
		return []string{"sudo", "apt", "purge", "-y"}
	} else {
		return []string{"sudo", "yum", "remove", "-y"}
	}
}
