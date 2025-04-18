// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

func RetrieveTokenFromFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("token file path is empty")
	}

	slog.Debug("Reading token from file", "path", path)
	var keyVal string
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read token from file: %w", err)
	}

	keyBytes = bytes.TrimSpace(keyBytes)
	keyBytes = bytes.TrimRight(keyBytes, "\n")
	keyVal = string(keyBytes)

	if keyVal == "" {
		return "", errors.New("failed to load token, token file is empty")
	}

	return keyVal, nil
}
