// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenCredentials_GetRequestMetadata(t *testing.T) {
	token := "test_token"
	id := "1234-5678-9012"
	credentials := &PerRPCCredentials{
		Token: token,
		ID: id,
	}

	metadata, err := credentials.GetRequestMetadata(context.TODO())
	require.NoError(t, err)

	expectedMetadata := map[string]string{
		TokenKey: token,
		UUID:     id,
	}

	for key, value := range expectedMetadata {
		assert.Equal(t, metadata[key], value)
	}
}

func TestTokenCredentials_RequireTransportSecurity(t *testing.T) {
	credentials := &PerRPCCredentials{}
	requireSecurity := credentials.RequireTransportSecurity()
	assert.True(t, requireSecurity)
}
