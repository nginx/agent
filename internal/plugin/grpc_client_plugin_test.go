// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrpcClient_Init(t *testing.T) {
	grpcClient := NewGrpcClient(nil, nil)
	assert.NotNil(t, grpcClient.messagePipe)
}
