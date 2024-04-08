// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package backoff

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

var invocations = 0

func TestWaitUntil(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		operation       backoff.Operation
		context         context.Context
		initialInterval time.Duration
		maxInterval     time.Duration
		maxElapsedTime  time.Duration
		expectedError   bool
	}{
		{
			name: "positive test",
			operation: func() error {
				return nil
			},
			context:         context.Background(),
			initialInterval: 10 * time.Microsecond,
			maxInterval:     100 * time.Microsecond,
			maxElapsedTime:  1000 * time.Microsecond,
			expectedError:   false,
		},
		{
			name: "error test",
			operation: func() error {
				return errors.New("error")
			},
			context:         context.Background(),
			initialInterval: 10 * time.Microsecond,
			maxInterval:     100 * time.Microsecond,
			maxElapsedTime:  1000 * time.Microsecond,
			expectedError:   true,
		},
		{
			name: "30ms timeout test",
			operation: func() error {
				return errors.New("timeout occurred")
			},
			context:         context.Background(),
			initialInterval: 10 * time.Millisecond,
			maxInterval:     10 * time.Millisecond,
			maxElapsedTime:  30 * time.Millisecond,
			expectedError:   true,
		},
		{
			name: "return after 3 retries",
			operation: func() error {
				invocations++

				if invocations > 3 {
					return nil
				}

				return errors.New("error")
			},
			context:         context.Background(),
			initialInterval: 1 * time.Millisecond,
			maxInterval:     10 * time.Millisecond,
			maxElapsedTime:  300 * time.Millisecond,
			expectedError:   false,
		},
	}

	for _, test := range tests {
		invocations = 0
		settings := &Settings{
			InitialInterval:     test.initialInterval,
			MaxInterval:         test.maxInterval,
			MaxElapsedTime:      test.maxElapsedTime,
			RandomizationFactor: RandomizationFactor,
			Multiplier:          Multiplier,
		}
		result := WaitUntil(test.context, settings, test.operation)

		if test.expectedError {
			assert.Errorf(t, result, test.name)
		} else {
			assert.NoErrorf(t, result, test.name)
		}
	}
}
