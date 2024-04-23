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

func CreateTestIDs(t testing.TB) (correlationID, instanceID uuid.UUID) {
	t.Helper()
	correlationID, err := uuid.Parse("1a968ddd-ef9b-4ad1-97a4-e4590467bcf7")
	require.NoError(t, err)
	instanceID, err = uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	return correlationID, instanceID
}
