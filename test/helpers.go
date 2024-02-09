// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RemoveFileWithErrorCheck(t *testing.T, fileName string) {
	t.Helper()
	err := os.Remove(fileName)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on os.Remove of file %s", fileName))
	}
}
