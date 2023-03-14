/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"os"
	"time"
)

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

func EnableWritePermissionForSocket(path string) error {
	timeout := time.After(time.Second * 1)
	var lastError error
	for {
		select {
		case <-timeout:
			return lastError
		default:
			lastError = os.Chmod(path, 0660)
			if lastError == nil {
				return nil
			}
		}
		<-time.After(time.Microsecond * 100)
	}
}
