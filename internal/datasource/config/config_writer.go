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
		Write(ctx context.Context, filesURL, tenantID, instanceID string) (skippedFiles CacheContent, err error)
		Complete(ctx context.Context) (err error)
		SetConfigClient(configClient client.ConfigClientInterface)
		Rollback(ctx context.Context, skippedFiles CacheContent, filesURL, tenantID, instanceID string) error
	}

	ConfigWriter struct {
		configClient       client.ConfigClientInterface
		allowedDirectories []string
		fileCache          FileCacheInterface
		currentFileCache   CacheContent
	}
)

func NewConfigWriter(agentConfig *config.Config, fileCache FileCacheInterface,
) (*ConfigWriter, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("failed to create config writer, agent config is nil")
	}

	configClient := client.NewHTTPConfigClient(agentConfig.Client.Timeout)

	return &ConfigWriter{
		configClient:       configClient,
		allowedDirectories: agentConfig.AllowedDirectories,
		fileCache:          fileCache,
	}, nil
}

func (cw *ConfigWriter) SetConfigClient(configClient client.ConfigClientInterface) {
	cw.configClient = configClient
}

func (cw *ConfigWriter) Rollback(ctx context.Context, skippedFiles CacheContent, filesURL,
	tenantID, instanceID string,
) error {
	slog.DebugContext(ctx, "Rolling back NGINX config changes due to error")
	if cw.fileCache.CacheContent() == nil {
		return fmt.Errorf("error rolling back, no instance file cache found for instance %s", instanceID)
	}
	for key, value := range cw.fileCache.CacheContent() {
		if _, ok := skippedFiles[key]; ok {
			continue
		}

		err := cw.updateFile(ctx, value, filesURL, tenantID, instanceID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cw *ConfigWriter) Write(ctx context.Context, filesURL,
	tenantID, instanceID string,
) (skippedFiles CacheContent, err error) {
	slog.DebugContext(ctx, "Write nginx config", "files_url", filesURL)
	currentFileCache, skippedFiles := make(CacheContent), make(CacheContent)

	cacheContent, err := cw.fileCache.ReadFileCache(ctx)
	if err != nil {
		return nil, err
	}

	filesMetaData, err := cw.getFileMetaData(ctx, filesURL, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	for _, fileData := range filesMetaData.GetFiles() {
		if cacheContent != nil && !doesFileRequireUpdate(cacheContent, fileData) {
			slog.DebugContext(
				ctx,
				"Skipping file as latest version is already on disk",
				"file_path", fileData.GetPath(),
			)
			currentFileCache[fileData.GetPath()] = cacheContent[fileData.GetPath()]
			skippedFiles[fileData.GetPath()] = fileData

			continue
		}
		slog.DebugContext(
			ctx,
			"Updating file, latest version not on disk",
			"file_path", fileData.GetPath(),
		)
		if updateErr := cw.updateFile(ctx, fileData, filesURL, tenantID, instanceID); updateErr != nil {
			return nil, updateErr
		}
		currentFileCache[fileData.GetPath()] = fileData
	}

	cw.currentFileCache = currentFileCache

	if cacheContent != nil {
		err = cw.removeFiles(ctx, currentFileCache, cacheContent)
	}

	return skippedFiles, err
}

func (cw *ConfigWriter) removeFiles(ctx context.Context, currentFileCache, fileCache CacheContent) error {
	for _, file := range fileCache {
		if _, ok := currentFileCache[file.GetPath()]; !ok {
			slog.DebugContext(ctx, "Removing file", "file_path", file.GetPath())
			if err := os.Remove(file.GetPath()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error removing file: %s error: %w", file.GetPath(), err)
			}
		}
	}

	return nil
}

func (cw *ConfigWriter) getFileMetaData(ctx context.Context, filesURL, tenantID, instanceID string,
) (filesMetaData *instances.Files, err error) {
	filesMetaData, err = cw.configClient.GetFilesMetadata(ctx, filesURL, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error getting files metadata from %s: %w", filesURL, err)
	}

	if len(filesMetaData.GetFiles()) == 0 {
		slog.DebugContext(ctx, "No file metadata for instance", "instance_id", instanceID)
		return nil, fmt.Errorf("error getting files metadata, no metadata exists for instance: %s", instanceID)
	}

	return filesMetaData, nil
}

func (cw *ConfigWriter) updateFile(ctx context.Context, fileData *instances.File,
	filesURL, tenantID, instanceID string,
) error {
	if !cw.isFilePathValid(ctx, fileData.GetPath()) {
		return fmt.Errorf("invalid file path: %s", fileData.GetPath())
	}
	fileDownloadResponse, fetchErr := cw.configClient.GetFile(ctx, fileData, filesURL, tenantID, instanceID)
	if fetchErr != nil {
		return fmt.Errorf("error getting file data from %s: %w", filesURL, fetchErr)
	}

	fetchErr = writeFile(ctx, fileDownloadResponse.GetFileContent(), fileDownloadResponse.GetFilePath())

	if fetchErr != nil {
		return fmt.Errorf("error writing to file %s: %w", fileDownloadResponse.GetFilePath(), fetchErr)
	}

	return nil
}

func (cw *ConfigWriter) Complete(ctx context.Context) error {
	slog.DebugContext(ctx, "Completing config apply")
	err := cw.fileCache.UpdateFileCache(ctx, cw.currentFileCache)
	if err != nil {
		return fmt.Errorf("error updating cache.json to %w", err)
	}

	return nil
}

func writeFile(ctx context.Context, fileContent []byte, filePath string) error {
	slog.DebugContext(ctx, "Writing to file", "file_path", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.DebugContext(ctx, "File does not exist, creating new file", "file_path", filePath)
		err = os.MkdirAll(path.Dir(filePath), filePermissions)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(filePath), err)
		}
	}

	err := os.WriteFile(filePath, fileContent, filePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}
	slog.DebugContext(ctx, "Content written to file", "file_path", filePath)

	return nil
}

func (cw *ConfigWriter) isFilePathValid(ctx context.Context, filePath string) (validPath bool) {
	slog.DebugContext(ctx, "Checking is file path is valid")
	if filePath == "" || strings.HasSuffix(filePath, "/") {
		return false
	}
	for _, dir := range cw.allowedDirectories {
		if strings.HasPrefix(filePath, dir) {
			return true
		}
	}

	slog.DebugContext(ctx, "File not in allowed directories", "path", filePath)

	return false
}

func doesFileRequireUpdate(fileCache CacheContent, fileData *instances.File) (updateRequired bool) {
	if len(fileCache) > 0 {
		fileOnSystem, ok := fileCache[fileData.GetPath()]
		if !ok {
			return true
		}

		return ok && fileOnSystem.GetLastModified().AsTime().Before(fileData.GetLastModified().AsTime())
	}

	return false
}
