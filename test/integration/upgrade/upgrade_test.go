package upgrade

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/test/integration/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	maxFileSize    = 70000000 // Max Size: 70 MB
	maxUpgradeTime = 30 * time.Second
)

var (
	osRelease  = os.Getenv("OS_RELEASE")
	osArch     = os.Getenv("ARCH")
	serverHost = map[string]string{
		"NGINX_AGENT_SERVER_HOST": "127.0.0.1",
	}

	expectedUpgradeLogMsgs = map[string]string{
		"UpgradeFoundNginxAgent": "Found nginx-agent /usr/bin/nginx-agent",
		"UpgradeAgentSuccess":    "NGINX Agent package has been successfully installed.",
		"UpgradeAgentStartCmd":   "sudo systemctl start nginx-agent",
	}
)

func TestUpgradeV2ToV3(t *testing.T) {
	log.Info("testing agent upgrade to v3")
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
		ServerHost:           serverHost,
	}

	testContainer := utils.StartContainer(
		ctx,
		t,
		containerNetwork,
		params,
	)

	// upgrade the agent to v3, check the upgrade time and verify the logs
	verifyAgentUpgrade(ctx, t, testContainer)

	// check the output of nginx-agent --version
	verifyAgentVersion(ctx, t, testContainer)

	// check the size of the upgraded agent package
	verifyAgentPackageSize(ctx, t, testContainer)

	// validate the nginx-agent config is upgraded and correct
	verifyAgentConfigFile(ctx, t, testContainer)

	log.Info("finished testing agent upgrade to v3")
}

func verifyAgentUpgrade(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	upgradeTime, cmdOut := upgradeAgent(ctx, t, testContainer)

	assert.LessOrEqual(t, upgradeTime, maxUpgradeTime)
	t.Log("upgrade time:", upgradeTime)

	upgradeLog, err := io.ReadAll(cmdOut)
	require.NoError(t, err)
	t.Log("upgrade log:", string(upgradeLog))

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
	t.Log("agent version:", string(agentVersion))
}

func verifyAgentPackageSize(ctx context.Context, t *testing.T, testContainer testcontainers.Container) {
	var packageSizeCmd []string

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		packageSizeCmd = []string{"dpkg-query", "-W", "--showformat=${Installed-Size}", "nginx-agent"}
	} else if strings.Contains(osRelease, "alpine") {
		packageSizeCmd = []string{"apk", "info", "-f", "nginx-agent", "--size"}
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

	// convert apt package size from KB to bytes
	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		packageSize *= 1024
	}

	assert.LessOrEqual(t, packageSize, maxFileSize)
	t.Log("package size:", packageSize)
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

	assert.Equal(t, string(expectedConfig), string(agentConfig))
	t.Log("agent config:", string(agentConfig))
}

func upgradeAgent(ctx context.Context, t *testing.T, testContainer testcontainers.Container) (time.Duration, io.Reader) {
	var updatePkgCmd []string
	var upgradeAgentCmd []string
	officialDebPackage := "./nginx-agent_3.2.1~bookworm_" + osArch + ".deb"

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		updatePkgCmd = []string{"apt-get", "update"}
		if os.Getenv("GITHUB_JOB") == "integration-tests" {
			upgradeAgentCmd = []string{"apt-get", "install", "-y", "--only-upgrade", "nginx-agent", "-o", "Dpkg::Options::=--force-confold"}
		} else {
			upgradeAgentCmd = []string{"apt-get", "install", "-y", officialDebPackage, "-o", "Dpkg::Options::=--force-confold"}
		}

	} else if strings.Contains(osRelease, "alpine") {
		updatePkgCmd = []string{"apk", "update"}
		upgradeAgentCmd = []string{"apk", "add", "nginx-agent=3.2.1"}
	} else {
		updatePkgCmd = []string{"yum", "-y", "makecache"}
		upgradeAgentCmd = []string{"yum", "update", "-y", "nginx-agent"}
	}

	exitCode, _, err := testContainer.Exec(ctx, updatePkgCmd)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	start := time.Now()

	exitCode, cmdOut, err := testContainer.Exec(ctx, upgradeAgentCmd)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	return time.Since(start), cmdOut
}
