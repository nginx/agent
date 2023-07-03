package features

import (
	"context"
	"io"
	"testing"

	"github.com/nginx/agent/test/integration/utils"
	"github.com/stretchr/testify/assert"
)

func TestFeatures_EnableDisable(t *testing.T) {
	testContainer := utils.SetupTestContainerWithAgent(t)

	utils.TestAgentHasNoErrorLogs(t, testContainer)

	exitCode, agentLogFile, err := testContainer.Exec(context.Background(), []string{"cat", "/var/log/nginx-agent/agent.log"})
	assert.NoError(t, err, "agent log file not found")
	assert.Equal(t, 0, exitCode)

	agentLogContent, err := io.ReadAll(agentLogFile)

	assert.NoError(t, err, "agent log file could not be read")
	assert.NotEmpty(t, agentLogContent, "agent log file empty")
	assert.Contains(t, string(agentLogContent), "level=info msg=\"OneTimeRegistration initializing\"", "agent log file contains OneTimeRegistration")
	assert.Contains(t, string(agentLogContent), "level=info msg=\"Metrics initializing\"", "agent log file contains OneTimeRegistration")
	assert.NotContains(t, string(agentLogContent), "level=info msg=\"DataPlaneStatus initializing\"", "agent log file contains OneTimeRegistration")

}
