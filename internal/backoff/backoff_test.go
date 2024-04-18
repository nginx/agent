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
	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestWaitUntil(t *testing.T) {
	t.Parallel()
	invocations := 0
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
		settings := &config.CommonSettings{
			InitialInterval:     test.initialInterval,
			MaxInterval:         test.maxInterval,
			MaxElapsedTime:      test.maxElapsedTime,
			RandomizationFactor: config.DefBackoffRandomizationFactor,
			Multiplier:          config.DefBackoffMultiplier,
		}
		result := WaitUntil(test.context, settings, test.operation)

		if test.expectedError {
			assert.Errorf(t, result, test.name)
		} else {
			assert.NoErrorf(t, result, test.name)
		}
	}
}

func TestWaitUntilWithData(t *testing.T) {
	t.Parallel()
	invocations := -1
	tests := []struct {
		name            string
		operation       backoff.OperationWithData[int]
		context         context.Context
		initialInterval time.Duration
		maxInterval     time.Duration
		maxElapsedTime  time.Duration
		expectedError   bool
	}{
		{
			name: "positive test",
			operation: func() (int, error) {
				return 0, nil
			},
			context:         context.Background(),
			initialInterval: 10 * time.Microsecond,
			maxInterval:     100 * time.Microsecond,
			maxElapsedTime:  1000 * time.Microsecond,
			expectedError:   false,
		},
		{
			name: "error test",
			operation: func() (int, error) {
				return 0, errors.New("error")
			},
			context:         context.Background(),
			initialInterval: 10 * time.Microsecond,
			maxInterval:     100 * time.Microsecond,
			maxElapsedTime:  1000 * time.Microsecond,
			expectedError:   true,
		},
		{
			name: "30ms timeout test",
			operation: func() (int, error) {
				return 0, errors.New("timeout occurred")
			},
			context:         context.Background(),
			initialInterval: 10 * time.Millisecond,
			maxInterval:     10 * time.Millisecond,
			maxElapsedTime:  30 * time.Millisecond,
			expectedError:   true,
		},
		{
			name: "return after 3 retries",
			operation: func() (int, error) {
				invocations++

				if invocations > 3 {
					return invocations, nil
				}

				return 0, errors.New("error")
			},
			context:         context.Background(),
			initialInterval: 1 * time.Millisecond,
			maxInterval:     10 * time.Millisecond,
			maxElapsedTime:  300 * time.Millisecond,
			expectedError:   false,
		},
	}

	for _, test := range tests {
		settings := &config.CommonSettings{
			InitialInterval:     test.initialInterval,
			MaxInterval:         test.maxInterval,
			MaxElapsedTime:      test.maxElapsedTime,
			RandomizationFactor: config.DefBackoffRandomizationFactor,
			Multiplier:          config.DefBackoffMultiplier,
		}
		result, err := WaitUntilWithData(test.context, settings, test.operation)

		assert.Equal(t, test.expectedError, err != nil)

		if test.expectedError {
			assert.Errorf(t, err, test.name)
		} else {
			assert.Greater(t, result, -1)
			assert.NoErrorf(t, err, test.name)
		}
	}
}

func TestContext(t *testing.T) {
	settings := &config.CommonSettings{
		InitialInterval:     10 * time.Millisecond,
		MaxInterval:         10 * time.Millisecond,
		MaxElapsedTime:      10 * time.Millisecond,
		RandomizationFactor: config.DefBackoffRandomizationFactor,
		Multiplier:          config.DefBackoffMultiplier,
	}

	backoffCtx := Context(context.Background(), settings)

	assert.NotEmpty(t, backoffCtx)
}
