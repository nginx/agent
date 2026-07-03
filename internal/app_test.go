// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApp_ConfigFileMissing(t *testing.T) {
	app := NewApp("1234", "1.2.3")

	err := app.Run(t.Context())

	require.Error(t, err, "app.Run must propagate the config-not-found error, not return nil")
}
