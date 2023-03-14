/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Name string `mapstructure:"name"`
}

func Test_DecodeConfig(t *testing.T) {
	// Valid input
	input := map[string]string{"name": "test-name"}
	output, err := DecodeConfig[*TestConfig](input)
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "test-name", output.Name)

	// Invalid input
	output, err = DecodeConfig[*TestConfig]("invalid")
	require.Error(t, err)
	assert.Nil(t, output)
}
