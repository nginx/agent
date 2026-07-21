// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package backoff

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v7"
	"github.com/nginx/agent/v3/internal/config"
)

// WaitUntil Implementation of backoff operations that increases the back off period for each retry attempt using
// a randomization function that grows exponentially.
//
// This is calculated using the following formula:
//
//	  randomized_interval =
//		  retry_interval * (random value in range [1 - randomization_factor, 1 + randomization_factor])
//
// The values range between the randomization factor percentage below and above the retry interval.
// For example, using 2 seconds as the base retry interval and 0.5 as the randomization factor,
// the actual back off period used in the next retry attempt will be between 1 and 3 seconds.
//
// NOTE: max_interval caps the retry_interval and not the randomized_interval.
//
// If the time elapsed since a backoff instance is created goes past the max_elapsed_time then the method
// starts returning stop.
//
// Example: The default retry_interval is .5 seconds, default randomization_factor is 0.5, default
// multiplier is 1.5 and the default max_interval is 1 minute. For 10 tries the sequence will be
// (values in seconds) and assuming we go over the max_elapsed_time on the 10th try:
//
//	request#     retry_interval     randomized_interval
//	1             0.5                [0.25,   0.75]
//	2             0.75               [0.375,  1.125]
//	3             1.125              [0.562,  1.687]
//	4             1.687              [0.8435, 2.53]
//	5             2.53               [1.265,  3.795]
//	6             3.795              [1.897,  5.692]
//	7             5.692              [2.846,  8.538]
//	8             8.538              [4.269, 12.807]
//	9            12.807              [6.403, 19.210]
//	10           19.210              {stop}
//
// Information from https://pkg.go.dev/github.com/cenkalti/backoff/v4#section-readme
// RetryOptions builds a slice of backoff.RetryOption from the config.BackOff
// settings and the context. Call sites can pass the returned slice to backoff.Retry as
// variadic options.
//
// If the context has a deadline, the elapsed time limit is set to the remaining
// time until that deadline (if it's less restrictive than the config-based
// MaxElapsedTime). This mirrors the v4 API behavior where WithContext would
// respect the context timeout. If the context has no deadline, the config-based
// MaxElapsedTime is used.
func RetryOptions(ctx context.Context, backoffSettings *config.BackOff) []backoff.RetryOption {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = backoffSettings.InitialInterval
	eb.MaxInterval = backoffSettings.MaxInterval
	eb.RandomizationFactor = backoffSettings.RandomizationFactor
	eb.Multiplier = backoffSettings.Multiplier

	maxElapsedTime := backoffSettings.MaxElapsedTime
	if deadline, ok := ctx.Deadline(); ok {
		timeUntilDeadline := time.Until(deadline)
		// Use the smaller of the two timeouts, but only if deadline is positive
		if timeUntilDeadline > 0 && (maxElapsedTime == 0 || timeUntilDeadline < maxElapsedTime) {
			maxElapsedTime = timeUntilDeadline
		}
	}

	return []backoff.RetryOption{
		backoff.WithBackOff(eb),
		backoff.WithMaxElapsedTime(maxElapsedTime),
	}
}

// WaitUntil retries a no-result operation until it succeeds, a permanent
// error is returned, or the retry options elapse. It adapts the operation to
// the generic backoff.Retry API.
func WaitUntil(
	ctx context.Context,
	backoffSettings *config.BackOff,
	operation func() error,
) error {
	retryOpts := RetryOptions(ctx, backoffSettings)
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		return struct{}{}, operation()
	}, retryOpts...)

	return err
}

// WaitUntilWithData Implementation of backoff operations that increases the back off period for each retry
// attempt using a randomization function that grows exponentially. This does not allow for parameters.
//
//nolint:ireturn // must return an interface
func WaitUntilWithData[T any](
	ctx context.Context,
	backoffSettings *config.BackOff,
	operation backoff.Operation[T],
) (T, error) {
	return backoff.Retry[T](ctx, operation, RetryOptions(ctx, backoffSettings)...)
}
