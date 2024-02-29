// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProcesses(t *testing.T) {
	p, err := GetProcesses()
	require.NoError(t, err)

	assert.NotEmpty(t, p)
}
