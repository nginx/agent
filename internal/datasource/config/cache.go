// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/nginx/agent/v3/api/grpc/instances"
)

const (
	cacheLocation = "/var/lib/nginx-agent/config/%s/cache.json"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . FileCacheInterface

type (
	FileCacheInterface interface {
		UpdateFileCache(cache CacheContent) error
		ReadFileCache() (CacheContent, error)
		SetCachePath(cachePath string)
		CacheContent() CacheContent
	}

	FileCache struct {
		cacheContent CacheContent
		CachePath    string
	}

	// map of files with filepath as key
	CacheContent = map[string]*instances.File
)

func NewFileCache(instanceID string) *FileCache {
	cachePath := fmt.Sprintf(cacheLocation, instanceID)

	return &FileCache{
		CachePath: cachePath,
	}
}

func (f *FileCache) ReadFileCache() (fileCache CacheContent, err error) {
	slog.Debug("Reading file cache")
	fileCache = make(CacheContent)

	_, statErr := os.Stat(f.CachePath)

	if statErr != nil {
		if os.IsNotExist(statErr) {
			return fileCache, fmt.Errorf("cache.json does not exist %s: %w", f.CachePath, statErr)
		}

		return fileCache, fmt.Errorf("error reading cache.json %s: %w", f.CachePath, statErr)
	}

	cacheData, err := os.ReadFile(f.CachePath)
	if err != nil {
		return fileCache, fmt.Errorf("error reading file cache.json %s: %w", f.CachePath, err)
	}
	err = json.Unmarshal(cacheData, &fileCache)
	if err != nil {
		return fileCache, fmt.Errorf("error unmarshalling data from cache.json %s: %w", f.CachePath, err)
	}

	return fileCache, err
}

func (f *FileCache) UpdateFileCache(cacheContent CacheContent) error {
	slog.Debug("Updating file cache")
	cachePath := f.CachePath
	cache, err := json.MarshalIndent(cacheContent, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cache data from %s: %w", cachePath, err)
	}

	err = writeFile(cache, cachePath)
	if err != nil {
		return fmt.Errorf("error writing cache to %s: %w", cachePath, err)
	}

	f.cacheContent = cacheContent

	return nil
}

func (f *FileCache) CacheContent() CacheContent {
	return f.cacheContent
}

func (f *FileCache) SetCachePath(cachePath string) {
	f.CachePath = cachePath
}
