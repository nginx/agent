// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoVersion(t *testing.T) {
	expected := "1.24.0"

	actual, err := GoVersion(t, 2)

	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestModuleVersion(t *testing.T) {
	expected := "1.25.0"

	actual, err := RequiredModuleVersion(t, "go.opentelemetry.io/otel/sdk", 2)

	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
