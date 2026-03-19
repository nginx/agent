/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package backoff

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	BACKOFF_JITTER     = 0.10
	BACKOFF_MULTIPLIER = backoff.DefaultMultiplier
)

type BackoffSettings struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
	Multiplier      float64
	Jitter          float64
}

func WaitUntil(
	ctx context.Context,
	backoffSettings BackoffSettings,
	operation backoff.Operation,
) error {
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = backoffSettings.InitialInterval
	exponentialBackoff.MaxInterval = backoffSettings.MaxInterval
	exponentialBackoff.MaxElapsedTime = backoffSettings.MaxElapsedTime
	exponentialBackoff.RandomizationFactor = backoffSettings.Jitter
	exponentialBackoff.Multiplier = backoffSettings.Multiplier

	expoBackoffWithContext := backoff.WithContext(exponentialBackoff, ctx)

	err := backoff.Retry(operation, expoBackoffWithContext)
	if err != nil {
		return err
	}

	return nil
}
