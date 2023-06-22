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

	// Check the environment variable $PACKAGE_REPO is set
	require.NotEmpty(t, AGENT_PACKAGE_REPO, "Environment variable $PACKAGE_REPO not set")

	testContainer := utils.SetupTestContainerWithoutAgent(t)

	ctx := context.Background()

	osReleaseContent, err := getOsReleaseContent(ctx, testContainer)
	require.NoError(t, err)

	err = installPrerequisites(testContainer, osReleaseContent)
	require.NoError(t, err, "failed to install prerequisites")

	err = downloadAndImportGPGKey(testContainer, osReleaseContent)
	require.NoError(t, err, "failed to download and import gpg key")

	err = createRepoFile(testContainer, osReleaseContent, AGENT_PACKAGE_REPO)
	require.NoError(t, err, "failed to create nginx-agent repo file in container")

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		err := updateRepo(testContainer, osReleaseContent)
		require.NoError(t, err, "failed to update repo packages cache")
	}

	// TODO: Check the file size is less than or equal 20MB

	// Install Agent inside container and record installation time/install output
	installLog, installTime, err := installAgent(ctx, testContainer, osReleaseContent)
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
	uninstallLog, err := uninstallAgent(ctx, testContainer, osReleaseContent)
	require.NoError(t, err)

	// Check uninstall output
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
	// Get OS to create uninstall cmd
	uninstallCmd := createUninstallCommand(osReleaseContent)

	// Start agent uninstall and capture uninstall output
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

func installPrerequisites(testContainer *testcontainers.DockerContainer, osReleaseContent string) error {
	var preReqCmd []string

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		preReqCmd = []string{"apt-get", "install", "-y", "curl", "gnupg2", "ca-certificates", "lsb-release", "ubuntu-keyring"}
	} else if strings.Contains(osReleaseContent, "alpine") {
		preReqCmd = []string{"apk", "add", "openssl", "curl", "ca-certificates"}
	} else {
		preReqCmd = []string{"yum", "install", "yum-utils"}
	}

	exitCode, out, err := testContainer.Exec(context.Background(), preReqCmd)
	stdOutStdErr, stdErr := io.ReadAll(out)
	if stdErr != nil {
		return fmt.Errorf("failed to read prerequisites cmd output: %v", stdErr)
	}
	if err != nil {
		return fmt.Errorf("failed to install prerequisites: %v\n%s", err, stdOutStdErr)
	}
	if exitCode != 0 {
		return fmt.Errorf("unexpected error code installing prerequisites. Expected 0, got: %v\n%s", exitCode, stdOutStdErr)
	}
	return nil
}

func downloadAndImportGPGKey(testContainer *testcontainers.DockerContainer, osReleaseContent string) error {
	var preReqCmd []string

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		preReqCmd = []string{"curl", "https://nginx.org/keys/nginx_signing.key", "|", "gpg", "--dearmor", "|",
			"tee", "/usr/share/keyrings/nginx-archive-keyring.gpg"}
	} else if strings.Contains(osReleaseContent, "alpine") {
		preReqCmd = []string{"curl", "-o", "/tmp/nginx_signing.rsa.pub", "https://nginx.org/keys/nginx_signing.rsa.pub"}
	} else {
		return nil // no GPG key required. yum install will fetch it
	}

	exitCode, out, err := testContainer.Exec(context.Background(), preReqCmd)
	stdOutStdErr, stdErr := io.ReadAll(out)
	if stdErr != nil {
		return fmt.Errorf("failed to read gpg key import cmd output: %v", stdErr)
	}

	if err != nil {
		return fmt.Errorf("failed to install prerequisites: %v\n%s", err, stdOutStdErr)
	}
	if exitCode != 0 {
		return fmt.Errorf("unexpected error code installing prerequisites. Expected 0, got: %v\n%s", exitCode, stdOutStdErr)
	}
	return nil
}

func createRepoFile(testContainer *testcontainers.DockerContainer, osReleaseContent string, packageRepo string) error {
	var repoFileContent, repoFilePath string

	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		// TODO: Get release. lsb_release -cs`
		repoFilePath = "/etc/apt/sources.list.d/nginx-agent.list"
		repoFileContent = `deb http://packages.nginx.org/nginx-agent/ubuntu/ focal agent`

		if strings.HasPrefix(packageRepo, "https://") {
			aptConfigPath := "/etc/apt/apt.conf.d/90pkgs-nginx"

			aptConfigContent := `Acquire::https::pkgs.nginx.com::Verify-Peer "true";
Acquire::https::pkgs.nginx.com::Verify-Host "true";
Acquire::https::pkgs.nginx.com::SslCert     "/etc/ssl/nginx/nginx-repo.crt";
Acquire::https::pkgs.nginx.com::SslKey      "/etc/ssl/nginx/nginx-repo.key";`

			err := testContainer.CopyToContainer(context.Background(), []byte(aptConfigContent), aptConfigPath, 0644)
			if err != nil {
				return fmt.Errorf("failed to copy repo config file to container: %v", err)
			}
		}

	} else if strings.Contains(osReleaseContent, "alpine") {
		repoFilePath = "/etc/yum.repos.d/nginx-agent.repo"

		if strings.HasPrefix(packageRepo, "https://") {
			repoFileContent = `[nginx-agent]
name=nginx agent repo
baseurl=https://pkgs.nginx.com/nginx-agent/centos/$releasever/$basearch/
sslclientcert=/etc/ssl/nginx/nginx-repo.crt
sslclientkey=/etc/ssl/nginx/nginx-repo.key
gpgcheck=0
enabled=1`
		} else {
			repoFileContent = `[nginx-agent]
name=nginx agent repo
baseurl=http://packages.nginx.org/nginx-agent/centos/$releasever/$basearch/
gpgcheck=1
enabled=1
gpgkey=https://nginx.org/keys/nginx_signing.key
module_hotfixes=true`
		}

	} else {
		repoFilePath = "/etc/apk/repositories"                                                    // TODO: Append not upsert
		repoFileContent = `@nginx-agent http://packages.nginx.org/nginx-agent/alpine/v$TODO/main` // TODO: Get alpine release
	}

	err := testContainer.CopyToContainer(context.Background(), []byte(repoFileContent), repoFilePath, 0644)
	if err != nil {
		return fmt.Errorf("failed to copy repo file to container: %v", err)
	}
	return nil
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
		return []string{"apk", "add", agentPackageName} // "--allow-untrusted"?
	} else {
		return []string{"yum", "install", "-y", agentPackageName}
	}
}

func createUninstallCommand(osReleaseContent string) []string {
	if strings.Contains(osReleaseContent, "UBUNTU") || strings.Contains(osReleaseContent, "Debian") {
		return []string{"apt", "purge", "-y", "nginx-agent"}
	} else if strings.Contains(osReleaseContent, "alpine") {
		return []string{"apk", "del", "nginx-agent"}
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
