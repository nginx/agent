// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestApp(t *testing.T) {
	app := NewApp("1234", "1.2.3")

	err := app.Run(context.Background())
	
	require.NoError(t, err)
}
