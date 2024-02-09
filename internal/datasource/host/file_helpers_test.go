// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	helpers "github.com/nginx/agent/v3/test"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGetPermissions(t *testing.T) {
	file, err := os.CreateTemp(".", "get_permissions_test.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	info, err := os.Stat(file.Name())
	require.NoError(t, err)

	permissions := GetPermissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}
