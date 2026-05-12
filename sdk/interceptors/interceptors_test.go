/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package interceptors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientInterceptor_ImplementsInterfaces(t *testing.T) {
	var ci interface{} = NewClientAuth(testUUID, testToken)

	_, okInterceptor := ci.(Interceptor)
	assert.True(t, okInterceptor, "clientInterceptor must implement Interceptor")

	_, okClient := ci.(ClientInterceptor)
	assert.True(t, okClient, "clientInterceptor must implement ClientInterceptor")
}
