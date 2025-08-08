// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmd(t *testing.T) {
	ctx := context.Background()
	ex := Exec{}

	output, err := ex.RunCmd(ctx, "/bin/ls")
	require.NoError(t, err)

	require.NotNil(t, output)
	assert.NotEmpty(t, output.String())
}

func TestFindExecutable(t *testing.T) {
	ex := Exec{}

	p, err := ex.FindExecutable("ls")
	require.NoError(t, err)

	assert.NotEmpty(t, p)
}
