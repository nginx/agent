package plugins

import (
	"testing"

	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestDataPlaneUpdateSubscriptions(t *testing.T) {
	pluginUnderTest := NewDataPlaneUpdate(tutils.GetMockAgentConfig(), tutils.NewMockEnvironment())
	assert.Equal(t, []string{}, pluginUnderTest.Subscriptions())

	pluginUnderTest.Close()
}

func TestDataPlaneUpdateInfo(t *testing.T) {
	pluginUnderTest := NewDataPlaneUpdate(tutils.GetMockAgentConfig(), tutils.NewMockEnvironment())
	info := pluginUnderTest.Info()
	assert.Equal(t, info.Name(), pluginUnderTest.Info().Name())
	assert.Equal(t, info.Version(), pluginUnderTest.Info().Version())

	pluginUnderTest.Close()
}
