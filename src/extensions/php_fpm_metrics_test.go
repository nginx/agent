/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions_test

import (
	"testing"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/stretchr/testify/assert"
)

func TestNewPhpFpmMetrics(t *testing.T) {
	_, err := extensions.NewPhpFpmMetrics(tutils.GetMockEnv(), &config.Config{})
	assert.NoError(t, err)
}

func TestPhpFpmMetrics_Info(t *testing.T) {
	plugin, err := extensions.NewPhpFpmMetrics(tutils.GetMockEnv(), tutils.GetMockAgentConfig())
	assert.NoError(t, err)
	assert.Equal(t, "php-fpm-metrics", plugin.Info().Name())
}
