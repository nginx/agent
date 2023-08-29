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

func TestGenerateID(t *testing.T) {
	result := GenerateID("%s_%s_%s", "/tmp", "/tmp/conf", "nim")
	assert.Equal(t, "5f17da2acca7a4429fd3070b039016360c32b49c0832ed2ced5751d3a1575488", result)
}
