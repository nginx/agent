// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package metricsutil_test

import (
	"testing"

	"github.com/nginx/agent/v3/internal/collector/metricsutil"
	"github.com/stretchr/testify/require"
)

func TestIncrease(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		current  int64
		previous int64
		expected int64
	}{
		{current: 10, previous: 5, expected: 5}, // normal increase
		{current: 5, previous: 5, expected: 0},  // no change
		{current: 0, previous: 4, expected: 0},  // reset
		{current: 3, previous: 10, expected: 3}, // reset then non-zero
	}

	for _, tc := range testcases {
		result := metricsutil.Increase(tc.current, tc.previous)
		require.Equal(t, tc.expected, result)
	}
}
