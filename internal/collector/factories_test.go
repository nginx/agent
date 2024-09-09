// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTelComponentFactories(t *testing.T) {
	factories, err := OTelComponentFactories()

	require.NoError(t, err, "OTelComponentFactories should not return an error")
	assert.NotNil(t, factories, "factories should not be nil")

	assert.Len(t, factories.Receivers, 4)
	assert.Len(t, factories.Processors, 20)
	assert.Len(t, factories.Exporters, 4)
	assert.Len(t, factories.Extensions, 3)
	assert.Empty(t, factories.Connectors)
}
