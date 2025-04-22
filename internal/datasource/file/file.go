// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

// ReadFromFile reads the contents from a file, trims the white space, trims newlines
// then returns the contents as a string
func ReadFromFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("failed to read file since file path is empty")
	}

	var content string
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read from file: %w", err)
	}

	contentBytes = bytes.TrimSpace(contentBytes)
	contentBytes = bytes.TrimRight(contentBytes, "\n")
	content = string(contentBytes)

	if content == "" {
		return "", errors.New("failed to read from file, file is empty")
	}

	return content, nil
}
