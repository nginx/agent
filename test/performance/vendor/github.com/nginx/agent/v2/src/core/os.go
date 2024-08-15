/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/backoff"
)

var chmodMutex sync.Mutex

// FileExists determines if the specified file given by the file path exists on the system.
// If the file does NOT exist on the system the bool will be false and the error will be nil,
// if the error is not nil then it's possible the file might exist but an error verifying it's
// existence has occurred.
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// FilesExists determines if the specified set of files exists on the system. If any of the files
// do NOT exist on the system the bool will be false and the error will be nil, if the error is
// not nil then it's possible the files might exist but an error verifying their existence has
// occurred.
func FilesExists(filePaths []string) (bool, error) {
	for _, filePath := range filePaths {
		fileExists, err := FileExists(filePath)
		if !fileExists || err != nil {
			return false, err
		}
	}

	return true, nil
}

// EnableWritePermissionForSocket attempts to set the write permissions for a socket file located at the specified path.
// The function continuously attempts the operation until either it succeeds or the timeout period elapses.
func EnableWritePermissionForSocket(ctx context.Context, path string) error {
	err := backoff.WaitUntil(ctx, backoff.BackoffSettings{
		InitialInterval: time.Microsecond * 100,
		MaxInterval:     time.Microsecond * 100,
		MaxElapsedTime:  time.Second * 1,
		Jitter:          backoff.BACKOFF_JITTER,
		Multiplier:      backoff.BACKOFF_MULTIPLIER,
	}, func() error {
		chmodMutex.Lock()
		lastError := os.Chmod(path, 0o660)
		chmodMutex.Unlock()
		return lastError
	})

	return err
}
