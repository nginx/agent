package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"github.com/nginx/agent/test/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	maxFileSize    = 60000000
	maxUpgradeTime = 30 * time.Second
)

var (
	osRelease              = os.Getenv("OS_RELEASE")
	expectedUpgradeLogMsgs = map[string]string{
		"UpgradeFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"UpgradeAgentSuccess":    "NGINX Agent package has been successfully installed.",
		"UpgradeAgentStartCmd":   "sudo systemctl start nginx-agent",
	}
)

func TestUpgradeToV3(t *testing.T) {
	ctx := context.Background()
	containerNetwork := utils.CreateContainerNetwork(ctx, t)

	nginxConf := "./nginx-oss.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConf = "./nginx-plus.conf"
	}

	params := &utils.Parameters{
		NginxAgentConfigPath: "./test_configs/nginx-agent.conf",
		NginxConfigPath:      nginxConf,
		LogMessage:           "NGINX Agent v",
	}

	testContainer := utils.StartContainer(
		ctx,
		t,
		containerNetwork,
		params,
	)

	// upgrade the agent to v3, check the upgrade time and verify the logs
	upgradeAgent(ctx, t, testContainer)

	// check the output of nginx-agent --version
	verifyAgentVersion(ctx, t, testContainer)

	// check the size of the upgraded agent package
	verifyAgentPackageSize(ctx, t, testContainer)

	// validate the nginx-agent config is upgraded and correct
	verifyAgentConfigFile(ctx, t, testContainer)
}

func upgradeAgent(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	updatePackageRepo(ctx, t, testContainer)
	upgradeCommand := createUpgradeCommand()

	start := time.Now()

	exitCode, cmdOut, err := testContainer.Exec(ctx, upgradeCommand)
	require.NoError(t, err)

	upgradeTime := time.Since(start)
	assert.LessOrEqual(t, upgradeTime, maxUpgradeTime)

	upgradeLog, err := io.ReadAll(cmdOut)
	fmt.Println(string(upgradeLog))
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	for _, logMsg := range expectedUpgradeLogMsgs {
		assert.Contains(t, string(upgradeLog), logMsg)
	}
}

func verifyAgentVersion(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	exitCode, agentVersionString, err := testContainer.Exec(ctx, []string{"nginx-agent", "--version"})
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	agentVersion, err := io.ReadAll(agentVersionString)
	require.NoError(t, err)
	assert.Contains(t, string(agentVersion), "nginx-agent version v3.")
}

func createUpgradeCommand() []string {
	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		return []string{"apt-get", "install", "-y", "--only-upgrade", "nginx-agent", "-o", "Dpkg::Options::=--force-confold"}
	}

	return []string{"yum", "update", "-y", "nginx-agent"}
}

func updatePackageRepo(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	var updateCmd []string

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		updateCmd = []string{"apt-get", "update"}
	} else {
		updateCmd = []string{"yum", "-y", "makecache"}
	}

	exitCode, _, err := testContainer.Exec(ctx, updateCmd)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func verifyAgentPackageSize(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	var packageSizeCmd []string

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		packageSizeCmd = []string{"dpkg-query", "-W", "--showformat=${Installed-Size}", "nginx-agent"}
	} else {
		packageSizeCmd = []string{"rpm", "-q", "--queryformat", "%{SIZE}", "nginx-agent"}
	}

	exitCode, packageSizeContent, err := testContainer.Exec(ctx, packageSizeCmd)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	packageSizeBytes, err := io.ReadAll(packageSizeContent)
	require.NoError(t, err)

	re := regexp.MustCompile(`\d+`)
	packageSizeStr := re.Find(packageSizeBytes)
	packageSize, err := strconv.Atoi(string(packageSizeStr))
	require.NoError(t, err)

	assert.LessOrEqual(t, packageSize, maxFileSize)
}

func verifyAgentConfigFile(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	agentConfigContent, err := testContainer.CopyFileFromContainer(ctx, "/etc/nginx-agent/nginx-agent.conf")
	require.NoError(t, err)

	agentConfig, err := io.ReadAll(agentConfigContent)
	require.NoError(t, err)

	expectedConfig, err := os.ReadFile("./test_configs/valid-v3-nginx-agent.conf")
	require.NoError(t, err)

	expectedConfig = bytes.TrimSpace(expectedConfig)
	agentConfig = bytes.TrimSpace(agentConfig)

	assert.Equal(t, expectedConfig, agentConfig)
}
