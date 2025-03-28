package features

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/nginx/agent/test/integration/utils"
	"github.com/stretchr/testify/assert"
)

func TestFeatures_NginxCountingEnabled(t *testing.T) {
	enabledFeatureLogs := []string{
		"level=info msg=\"NGINX Counter initializing", "level=info msg=\"MetricsThrottle initializing\"", "level=info msg=\"DataPlaneStatus initializing\"",
		"level=info msg=\"OneTimeRegistration initializing\"", "level=info msg=\"Metrics initializing\"",
	}
	disabledFeatureLogs := []string{"level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\""}

	ctx := context.Background()
	containerNetwork := utils.CreateContainerNetwork(ctx, t)

	nginxConf := "./nginx-oss.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConf = "./nginx-plus.conf"
	}

	params := &utils.Parameters{
		NginxAgentConfigPath: "./test_configs/nginx-agent-counting.conf",
		NginxConfigPath:      nginxConf,
		LogMessage:           "MetricsThrottle waiting for report ready",
	}

	testContainer := utils.StartContainer(
		ctx,
		t,
		containerNetwork,
		params,
	)

	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}
}

func TestFeatures_MetricsEnabled(t *testing.T) {
	enabledFeatureLogs := []string{"level=info msg=\"Metrics initializing\"", "level=info msg=\"MetricsThrottle initializing\"", "level=info msg=\"DataPlaneStatus initializing\""}
	disabledFeatureLogs := []string{"level=info msg=\"OneTimeRegistration initializing\"", "level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\""}

	ctx := context.Background()
	containerNetwork := utils.CreateContainerNetwork(ctx, t)

	nginxConf := "./nginx-oss.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConf = "./nginx-plus.conf"
	}

	params := &utils.Parameters{
		NginxAgentConfigPath: "./test_configs/nginx-agent-metrics.conf",
		NginxConfigPath:      nginxConf,
		LogMessage:           "MetricsThrottle waiting for report ready",
	}

	testContainer := utils.StartContainer(
		ctx,
		t,
		containerNetwork,
		params,
	)

	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}
}

func TestFeatures_ConfigEnabled(t *testing.T) {
	enabledFeatureLogs := []string{"level=info msg=\"DataPlaneStatus initializing\""}
	disabledFeatureLogs := []string{"level=info msg=\"Events initializing\"", "level=info msg=\"Agent API initializing\"", "level=info msg=\"Metrics initializing\"", "level=info msg=\"MetricsThrottle initializing\""}

	ctx := context.Background()
	containerNetwork := utils.CreateContainerNetwork(ctx, t)

	nginxConf := "./nginx-oss.conf"
	if os.Getenv("IMAGE_PATH") == "/nginx-plus/agent" {
		nginxConf = "./nginx-plus.conf"
	}

	params := &utils.Parameters{
		NginxAgentConfigPath: "./test_configs/nginx-agent-config.conf",
		NginxConfigPath:      nginxConf,
		LogMessage:           "DataPlaneStatus initializing",
	}

	testContainer := utils.StartContainer(
		ctx,
		t,
		containerNetwork,
		params,
	)

	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")

	for _, logLine := range enabledFeatureLogs {
		assert.Contains(t, string(agentLogContent), logLine, "agent log file does not contain enabled feature log")
	}

	for _, logLine := range disabledFeatureLogs {
		assert.NotContains(t, string(agentLogContent), logLine, "agent log file contains disabled feature log")
	}
}
