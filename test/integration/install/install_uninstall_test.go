package install

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

const (
	osReleasePath  = "/etc/os-release"
	maxFileSize    = int64(20000000)
	maxInstallTime = 30 * time.Second
)

var (
	AGENT_PACKAGE_FILENAME = os.Getenv("PACKAGE_NAME")
	AGENT_PACKAGE_FILEPATH = "/agent/build/" + AGENT_PACKAGE_FILENAME + ".deb"
	agentContainer         *testcontainers.DockerContainer
)

func setupTestContainer(t *testing.T) {
	comp, err := compose.NewDockerCompose("docker-compose.yml")
	assert.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		assert.NoError(t, comp.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal), "compose.Down()")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, comp.WithEnv(map[string]string{
		"PACKAGE_NAME": os.Getenv("PACKAGE_NAME"),
		"BASE_IMAGE":   os.Getenv("BASE_IMAGE"),
	},
	).Up(ctx, compose.Wait(true)), "compose.Up()")

	agentContainer, err = comp.ServiceContainer(context.TODO(), "agent")
	require.NoError(t, err)
}

// TestAgentManualInstallUninstall tests Agent Install and Uninstall.
// Verifies that agent installs with correct output and files.
// Verifies that agent uninstalls and removes all the files.
func TestAgentManualInstallUninstall(t *testing.T) {
	// Check the environment variable $PACKAGE_NAME is set
	require.NotEmpty(t, AGENT_PACKAGE_FILENAME, "Environment variable $PACKAGE_NAME not set")

	setupTestContainer(t)

	exitCode, o, err := agentContainer.Exec(context.TODO(), []string{"cat", osReleasePath})
	assert.NoError(t, err)
	osReleaseContent, err := io.ReadAll(o)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.NotEmpty(t, osReleaseContent, "os release file empty")

	expectedInstallLogMsgs := map[string]string{
		"InstallFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentToRunAs":    "nginx-agent will be configured to run as same user",
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

	// Check the agent package is present
	agentPkg, err := agentContainer.CopyFileFromContainer(context.TODO(), AGENT_PACKAGE_FILEPATH)
	assert.NoError(t, err)

	a, err := io.ReadAll(agentPkg)
	assert.NoError(t, err)

	f, err := os.CreateTemp("", "tmpfile-")
	assert.NoError(t, err)

	_, err = f.Write(a)
	assert.NoError(t, err)
	f.Close()

	file, err := os.Stat(f.Name())
	assert.NoError(t, err, "Error accessing package at:", AGENT_PACKAGE_FILEPATH)

	// Check the file size is less than or equal 20MB
	assert.LessOrEqual(t, file.Size(), maxFileSize)

	// Install Agent and record installation time/install output
	installTime, installLog := installAgent(t, agentContainer, string(osReleaseContent))

	// Check the install time under 30s
	assert.LessOrEqual(t, installTime, maxInstallTime)

	// Check install output
	for log, logMsg := range expectedInstallLogMsgs {
		if log == "InstallAgentToRunAs" && !nginxIsRunning() {
			continue // only expected if nginx is installed and running
		}
		assert.Contains(t, installLog, logMsg)
	}

	// Check nginx-agent config files were created.
	for _, path := range expectedAgentPaths {
		_, err = agentContainer.CopyFileFromContainer(context.TODO(), path)
		assert.NoError(t, err)
	}

	// Uninstall the agent package
	uninstallLog := uninstallAgent(t, agentContainer, string(osReleaseContent))

	// Check uninstall output
	for _, logMsg := range expectedUninstallLogMsgs {
		assert.Contains(t, uninstallLog, logMsg)
	}

	// Check nginx-agent config files were removed.
	for path := range expectedAgentPaths {
		_, err = agentContainer.CopyFileFromContainer(context.TODO(), path)
		assert.Error(t, err)
	}
}

// installAgent installs the agent returning total install time and install output
func installAgent(t *testing.T, container *testcontainers.DockerContainer, osReleaseContent string) (time.Duration, string) {
	// Get OS to create install cmd
	installCmd := createInstallCommand(osReleaseContent)

	// Start install timer
	start := time.Now()

	// Start agent installation and capture install output
	exitCode, cmdOut, err := container.Exec(context.TODO(), installCmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode, "expected exit code of 0")

	stdoutStderr, err := io.ReadAll(cmdOut)
	assert.NoError(t, err)

	elapsed := time.Since(start)

	return elapsed, string(stdoutStderr)
}

// uninstallAgent uninstall the agent returning output
func uninstallAgent(t *testing.T, container *testcontainers.DockerContainer, osReleaseContent string) string {
	// Get OS to create uninstall cmd
	uninstallCmd := createUninstallCommand(osReleaseContent)

	// Start agent uninstall and capture uninstall output
	exitCode, cmdOut, err := container.Exec(context.TODO(), uninstallCmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	stdoutStderr, err := io.ReadAll(cmdOut)
	assert.NoError(t, err)
	return string(stdoutStderr)
}

func createInstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"dpkg", "-i", AGENT_PACKAGE_FILEPATH}
	} else {
		return []string{"yum", "localinstall", AGENT_PACKAGE_FILEPATH}
	}
}

func createUninstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"apt", "purge", "-y", "nginx-agent"}
	} else {
		return []string{"yum", "remove", "-y", "nginx-agent"}
	}
}

func nginxIsRunning() bool {
	processes, _ := process.Processes()

	for _, process := range processes {
		name, _ := process.Name()
		if name == "nginx" {
			return true
		}
	}

	return false
}
