package sdk

import (
	"context"
	"errors"

	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

type testObj struct {
	name            string
	operation       backoff.Operation
	context         context.Context
	initialInterval time.Duration
	maxInterval     time.Duration
	maxElapsedTime  time.Duration
	expectedError   bool
}

var (
	invocations = 0
)

func TestBackOff(t *testing.T) {
	t.Parallel()
	tests := []testObj{
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
				invocations += 1

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
		result := WaitUntil(test.context, test.initialInterval, test.maxInterval, test.maxElapsedTime, test.operation)

		if test.expectedError {
			assert.Errorf(t, result, test.name)
		} else {
			assert.NoErrorf(t, result, test.name)
		}
	}

}
