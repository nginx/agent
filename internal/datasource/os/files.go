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

func WriteFile(fileContent []byte, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating new file", "file", filePath)
		err = os.MkdirAll(path.Dir(filePath), 0o750)
		if err != nil {
			return fmt.Errorf("error creating directory, directory: %v, error: %v", path.Dir(filePath), err)
		}
	}

	err := os.WriteFile(filePath, fileContent, 0o644)
	if err != nil {
		return fmt.Errorf("error writing to file, filePath: %v, error:%v", filePath, err)
	}
	slog.Debug("Content written to file", "filePath", filePath)

	return nil
}

func ReadInstanceCache(cachePath string) (map[string]*instances.File, error) {
	lastConfigApply := make(map[string]*instances.File)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return lastConfigApply, fmt.Errorf("previous config apply cache.json does not exist, cachePath: %v, error: %v", cachePath, err)
	}

	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		return lastConfigApply, fmt.Errorf("error reading file cache.json, cachePath: %v, error: %v", cachePath, err)
	}
	err = json.Unmarshal(cacheData, &lastConfigApply)
	if err != nil {
		return lastConfigApply, fmt.Errorf("error unmarshalling data from cache.json, cachePath: %v, error: %v", cachePath, err)
	}

	return lastConfigApply, err
}

func UpdateCache(currentConfigApply map[string]*instances.File, cachePath string) error {
	cache, err := json.MarshalIndent(currentConfigApply, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling cache data, cachePath: %v, error: %v", cachePath, err)
	}

	err = WriteFile(cache, cachePath)
	if err != nil {
		return fmt.Errorf("error writing cache, cachePath: %v, error: %v", cachePath, err)
	}

	return err
}
