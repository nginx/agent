/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	name := "plugin"
	version := "v0.0.1"
	info := NewInfo(name, version)

	assert.Equal(t, name, info.Name())
	assert.Equal(t, version, info.Version())
}
