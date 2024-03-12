// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"
)

const (
	instanceID = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
)

func CreateTestIDs(t testing.TB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	require.NoError(t, err)

	instanceID, err := uuid.Parse(instanceID)
	require.NoError(t, err)

	return tenantID, instanceID
}
