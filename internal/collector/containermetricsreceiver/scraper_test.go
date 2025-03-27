package containermetricsreceiver

import (
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"testing"
)

func TestScraper(t *testing.T) {

	cfg, ok := config.CreateDefaultConfig().(*config.Config)
	assert.True(t, ok)
	require.NoError(t, component.ValidateConfig(*cfg))

	_, err := newContainerScraper(receivertest.NewNopSettings(), cfg)
	require.NoError(t, err)
}
