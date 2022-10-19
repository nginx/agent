package plugins

import (
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestDataPlaneUpdateSubscriptions(t *testing.T) {
	pluginUnderTest := NewDataPlaneUpdate(tutils.GetMockAgentConfig(), tutils.NewMockNginxBinary(), tutils.NewMockEnvironment(), &proto.Metadata{MessageId: "1234"}, "1.0")
	assert.Equal(t, []string{core.AgentConfigChanged, core.NginxAppProtectDetailsGenerated}, pluginUnderTest.Subscriptions())

	pluginUnderTest.Close()
}

func TestDataPlaneUpdateInfo(t *testing.T) {
	pluginUnderTest := NewDataPlaneUpdate(tutils.GetMockAgentConfig(), tutils.NewMockNginxBinary(), tutils.NewMockEnvironment(), &proto.Metadata{MessageId: "1234"}, "1.0")
	info := pluginUnderTest.Info()
	assert.Equal(t, info.Name(), pluginUnderTest.Info().Name())
	assert.Equal(t, info.Version(), pluginUnderTest.Info().Version())

	pluginUnderTest.Close()
}
