// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
)

const (
	filePermissions = 0o600
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ConfigWriterInterface

type (
	ConfigWriterInterface interface {
		Write(ctx context.Context, filesURL, tenantID, instanceID string) (skippedFiles map[string]struct{}, err error)
		Complete() (err error)
		SetConfigClient(configClient client.ConfigClientInterface)
	}

	ConfigWriter struct {
		configClient       client.ConfigClientInterface
		allowedDirectories []string
		fileCache          FileCacheInterface
		currentFileCache   CacheContent
	}
)

func NewConfigWriter(agentConfig *config.Config, fileCache FileCacheInterface,
) *ConfigWriter {
	configClient := client.NewHTTPConfigClient(agentConfig.Client.Timeout)
	return &ConfigWriter{
		configClient:       configClient,
		allowedDirectories: agentConfig.AllowedDirectories,
		fileCache:          fileCache,
	}
}

func (cw *ConfigWriter) SetConfigClient(configClient client.ConfigClientInterface) {
	cw.configClient = configClient
}

func (cw *ConfigWriter) Write(ctx context.Context, filesURL,
	tenantID, instaneID string,
) (skippedFiles map[string]struct{}, err error) {
	currentFileCache := make(CacheContent)
	skippedFiles = make(map[string]struct{})
	cacheContent, err := cw.fileCache.ReadFileCache()
	if err != nil {
		slog.Warn("Failed to read file cache")
	}

	filesMetaData, err := cw.configClient.GetFilesMetadata(ctx, filesURL, tenantID, instaneID)
	if err != nil {
		return nil, fmt.Errorf("error getting files metadata from %s: %w", filesURL, err)
	}

	for _, fileData := range filesMetaData.GetFiles() {
		if !doesFileRequireUpdate(cacheContent, fileData) {
			slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.GetPath())
			currentFileCache[fileData.GetPath()] = cacheContent[fileData.GetPath()]
			skippedFiles[fileData.GetPath()] = struct{}{}

			continue
		}
		file, updateErr := cw.updateFile(ctx, fileData, filesURL, tenantID, instaneID)
		if updateErr != nil {
			slog.Debug("Update Error", "err", updateErr)
			continue
		}
		currentFileCache[fileData.GetPath()] = file
	}

	cw.currentFileCache = currentFileCache

	return skippedFiles, err
}

func (cw *ConfigWriter) updateFile(ctx context.Context, fileData *instances.File,
	filesURL, tenantID, instanceID string,
) (*instances.File, error) {
	if !cw.isFilePathValid(fileData.GetPath()) {
		return nil, fmt.Errorf("invalid file path: %s", fileData.GetPath())
	}
	fileDownloadResponse, fetchErr := cw.configClient.GetFile(ctx, fileData, filesURL, tenantID, instanceID)
	if fetchErr != nil {
		return nil, fmt.Errorf("error getting file data from %s: %w", filesURL, fetchErr)
	}

	fetchErr = writeFile(fileDownloadResponse.GetFileContent(), fileDownloadResponse.GetFilePath())

	if fetchErr != nil {
		return nil, fmt.Errorf("error writing to file %s: %w", fileDownloadResponse.GetFilePath(), fetchErr)
	}

	return &instances.File{
		Version:      fileData.GetVersion(),
		Path:         fileData.GetPath(),
		LastModified: fileData.GetLastModified(),
	}, nil
}

func (cw *ConfigWriter) Complete() error {
	err := cw.fileCache.UpdateFileCache(cw.currentFileCache)
	if err != nil {
		return fmt.Errorf("error updating cache to %s: %w", cw.fileCache.GetCachePath(), err)
	}

	return nil
}

func writeFile(fileContent []byte, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating new file", "file", filePath)
		err = os.MkdirAll(path.Dir(filePath), filePermissions)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(filePath), err)
		}
	}

	err := os.WriteFile(filePath, fileContent, filePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}
	slog.Debug("Content written to file", "filePath", filePath)

	return nil
}

func (cw *ConfigWriter) isFilePathValid(filePath string) (validPath bool) {
	if filePath == "" || strings.HasSuffix(filePath, "/") {
		return false
	}
	for _, dir := range cw.allowedDirectories {
		if strings.HasPrefix(filePath, dir) {
			return true
		}
	}

	return false
}

func doesFileRequireUpdate(fileCache CacheContent, fileData *instances.File) (updateRequired bool) {
	if fileCache == nil {
		return true
	}
	if len(fileCache) > 0 {
		fileOnSystem, ok := fileCache[fileData.GetPath()]
		if !ok {
			return true
		}

		return ok && fileOnSystem.GetLastModified().AsTime().Before(fileData.GetLastModified().AsTime())
	}

	return false
}
