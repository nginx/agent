// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serverType ServerType
		expected   string
	}{
		{"Test 1: command type", Command, "command"},
		{"Test 2: auxiliary type", Auxiliary, "auxiliary"},
		{"Test 3: unknown type", ServerType(99), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.serverType.String())
		})
	}
}
