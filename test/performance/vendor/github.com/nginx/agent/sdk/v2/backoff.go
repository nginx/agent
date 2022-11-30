/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	BACKOFF_JITTER     = 0.10
	BACKOFF_MULTIPLIER = backoff.DefaultMultiplier
)

func WaitUntil(
	ctx context.Context,
	initialInterval time.Duration,
	maxInterval time.Duration,
	maxElapsedTime time.Duration,
	operation backoff.Operation,
) error {
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = initialInterval
	exponentialBackoff.MaxInterval = maxInterval
	exponentialBackoff.MaxElapsedTime = maxElapsedTime
	exponentialBackoff.RandomizationFactor = BACKOFF_JITTER
	exponentialBackoff.Multiplier = BACKOFF_MULTIPLIER

	expoBackoffWithContext := backoff.WithContext(exponentialBackoff, ctx)

	err := backoff.Retry(backoff.Operation(operation), expoBackoffWithContext)
	if err != nil {
		return err
	}

	return nil
}
