// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package backoff

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	Jitter     = 0.10
	Multiplier = backoff.DefaultMultiplier
)

type Backoff[T any] struct{}

type Settings struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
	Multiplier      float64
	Jitter          float64
}

func WaitUntil(
	ctx context.Context,
	backoffSettings *Settings,
	operation backoff.Operation,
) error {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = backoffSettings.InitialInterval
	eb.MaxInterval = backoffSettings.MaxInterval
	eb.MaxElapsedTime = backoffSettings.MaxElapsedTime
	eb.RandomizationFactor = backoffSettings.Jitter
	eb.Multiplier = backoffSettings.Multiplier

	backoffWithContext := backoff.WithContext(eb, ctx)

	err := backoff.Retry(operation, backoffWithContext)
	if err != nil {
		return err
	}

	return nil
}
