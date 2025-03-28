// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package containermetricsreceiver

import (
	"testing"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestScraper(t *testing.T) {
	cfg, ok := config.CreateDefaultConfig().(*config.Config)
	assert.True(t, ok)
	require.NoError(t, component.ValidateConfig(*cfg))

	s := newContainerScraper(receivertest.NewNopSettings(), cfg)
	require.NotNil(t, s)
}
