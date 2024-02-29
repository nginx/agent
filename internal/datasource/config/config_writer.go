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
		Complete() (err error)
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
	slog.Debug("Rolling back NGINX config changes due to error")
	for key, value := range cw.fileCache.CacheContent() {
		if _, ok := skippedFiles[key]; ok {
			continue
		}

		_, err := cw.updateFile(ctx, value, filesURL, tenantID, instanceID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cw *ConfigWriter) Write(ctx context.Context, filesURL,
	tenantID, instanceID string,
) (skippedFiles CacheContent, err error) {
	slog.Info("Writing file")
	currentFileCache := make(CacheContent)
	skippedFiles = CacheContent{}

	cacheContent, _ := cw.fileCache.ReadFileCache()

	filesMetaData, err := cw.getFileMetaData(ctx, filesURL, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	for _, fileData := range filesMetaData.GetFiles() {
		if !doesFileRequireUpdate(cacheContent, fileData) {
			slog.Info("Skipping file as latest version is already on disk", "file_path", fileData.GetPath())
			currentFileCache[fileData.GetPath()] = cacheContent[fileData.GetPath()]
			skippedFiles[fileData.GetPath()] = fileData

			continue
		}
		slog.Info("Updating file, latest version not on disk", "file_path", fileData.GetPath())
		file, updateErr := cw.updateFile(ctx, fileData, filesURL, tenantID, instanceID)
		if updateErr != nil {
			slog.Info("Update Error", "err", updateErr)
			skippedFiles[fileData.GetPath()] = fileData
		} else {
			currentFileCache[fileData.GetPath()] = file
		}
	}

	cw.currentFileCache = currentFileCache

	slog.Info("Skipped Files in Write", "", skippedFiles)

	return skippedFiles, err
}

func (cw *ConfigWriter) getFileMetaData(ctx context.Context, filesURL, tenantID, instanceID string,
) (filesMetaData *instances.Files, err error) {
	filesMetaData, err = cw.configClient.GetFilesMetadata(ctx, filesURL, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("error getting files metadata from %s: %w", filesURL, err)
	}

	if len(filesMetaData.GetFiles()) == 0 {
		slog.Debug("No file metadata for instance", "instance_id", instanceID)
		return nil, fmt.Errorf("error getting files metadata, no metadata exists for instance: %s", instanceID)
	}

	return filesMetaData, nil
}

func (cw *ConfigWriter) updateFile(ctx context.Context, fileData *instances.File,
	filesURL, tenantID, instanceID string,
) (*instances.File, error) {
	if !cw.isFilePathValid(fileData.GetPath()) {
		slog.Info("Invalid File Path, Skipping file")
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
	slog.Debug("Completing config apply")
	err := cw.fileCache.UpdateFileCache(cw.currentFileCache)
	if err != nil {
		return fmt.Errorf("error updating cache.json to %w", err)
	}

	return nil
}

func writeFile(fileContent []byte, filePath string) error {
	slog.Debug("Writing to file", "file_path", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating new file", "file_path", filePath)
		err = os.MkdirAll(path.Dir(filePath), filePermissions)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(filePath), err)
		}
	}

	err := os.WriteFile(filePath, fileContent, filePermissions)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filePath, err)
	}
	slog.Debug("Content written to file", "file_path", filePath)

	return nil
}

func (cw *ConfigWriter) isFilePathValid(filePath string) (validPath bool) {
	slog.Debug("Checking is file path is valid")
	if filePath == "" || strings.HasSuffix(filePath, "/") {
		return false
	}
	for _, dir := range cw.allowedDirectories {
		if strings.HasPrefix(filePath, dir) {
			return true
		}
	}

	slog.Debug("file not in allowed directories:", "path", filePath)

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
