/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package os

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/nginx/agent/v3/api/grpc/instances"
)

// map of files with filepath as key
type FileCache = map[string]*instances.File

func WriteFile(fileContent []byte, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating new file", "file", filePath)
		err = os.MkdirAll(path.Dir(filePath), 0o750)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(filePath), err)
		}
	}

	err := os.WriteFile(filePath, fileContent, 0o644)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}
	slog.Debug("Content written to file", "filePath", filePath)

	return nil
}

func ReadInstanceCache(cachePath string) (FileCache, error) {
	previousFileCache := FileCache{}

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return previousFileCache, fmt.Errorf("cache.json does not exist %s: %w", cachePath, err)
	}

	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		return previousFileCache, fmt.Errorf("error reading file cache.json %s: %w", cachePath, err)
	}
	err = json.Unmarshal(cacheData, &previousFileCache)
	if err != nil {
		return previousFileCache, fmt.Errorf("error unmarshalling data from cache.json %s: %w", cachePath, err)
	}

	return previousFileCache, err
}

func UpdateCache(currentFileCache FileCache, cachePath string) error {
	cache, err := json.MarshalIndent(currentFileCache, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling cache data from %s: %w", cachePath, err)
	}

	err = WriteFile(cache, cachePath)
	if err != nil {
		return fmt.Errorf("error writing cache to %s: %w", cachePath, err)
	}

	return err
}
