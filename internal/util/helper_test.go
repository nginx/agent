/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUUID(t *testing.T) {
	result := GenerateUUID("%s_%s_%s", "test1", "test2", "test3")
	expected := "02be9e7f-a802-35d4-9e4a-6c677259a87d"
	assert.Equal(t, expected, result)
}
