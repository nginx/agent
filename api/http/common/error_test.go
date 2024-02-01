// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package common

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	err := &RequestError{StatusCode: http.StatusInternalServerError, Message: "something went wrong"}
	assert.Equal(t, "something went wrong", err.Error())
}
