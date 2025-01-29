// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessOperator(t *testing.T) {
	po := NewProcessOperator()
	assert.NotNil(t, po)
}

func TestProcessOperator_Processes(t *testing.T) {
	ctx := context.Background()
	po := &ProcessOperator{}
	got, err := po.Processes(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
}
