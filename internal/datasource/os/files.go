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
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
)

func WriteFile(fileContent []byte, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating")
		err = os.MkdirAll(path.Dir(filePath), 0o750)
		if err != nil {
			slog.Error("Error creating directory", "dir", path.Dir(filePath), "error", err)
			return err
		}
	}

	err := os.WriteFile(filePath, fileContent, 0o644)
	if err != nil {
		slog.Error("Error writing to file", "filePath", filePath, "error", err)
		return err
	}
	slog.Debug("Content written to file", "filePath", filePath)

	return nil
}

// Temp for testing
func UpdateNginxConfig(tenantID uuid.UUID, instanceID uuid.UUID, filesUrl string) error {
	cachePath := fmt.Sprintf("/var/lib/nginx-agent/config/%v/cache.json", instanceID.String())

	lastConfigApply, err := ReadCache(cachePath)
	if err != nil {
		slog.Error("Failed to read cache.json", "cachePath", cachePath, "error", err)
	}

	currentConfigApply, skippedFiles, err := UpdateInstanceConfig(lastConfigApply, filesUrl, tenantID)
	if err != nil {
		slog.Error("Failed to update config", "cachePath", cachePath, "error", err)
	}

	err = UpdateCache(currentConfigApply, cachePath)
	if err != nil {
		slog.Error("Failed to update cache.json", "cachePath", cachePath, "error", err)
	}

	slog.Info("Skipped Files", "files", skippedFiles)

	return err
}

func ReadCache(cachePath string) (map[string]*instances.File, error) {
	lastConfigApply := make(map[string]*instances.File)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		slog.Debug("previous config apply cache.json does not exist", "cachePath", cachePath, "error", err)
		return lastConfigApply, err
	}

	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		slog.Error("Unable to read file cache.json", "cachePath", cachePath, "error", err)
		return lastConfigApply, err
	}
	err = json.Unmarshal(cacheData, &lastConfigApply)
	if err != nil {
		slog.Error("Unable to unmarshal data from cache.json", "cachePath", cachePath, "error", err)
		return lastConfigApply, err
	}

	return lastConfigApply, err
}

func UpdateCache(currentConfigApply map[string]*instances.File, cachePath string) error {
	cache, err := json.MarshalIndent(currentConfigApply, "", "  ")
	if err != nil {
		slog.Error("Unable marshal cache data", "cachePath", cachePath, "error", err)
		return err
	}

	err = WriteFile(cache, cachePath)
	if err != nil {
		slog.Error("Unable to write cache", "cachePath", cachePath, "error", err)
		return err
	}

	return err
}

func UpdateInstanceConfig(lastConfigApply map[string]*instances.File, filesUrl string, tenantID uuid.UUID) (currentConfigApply map[string]*instances.File, skippedFiles map[string]struct{}, err error) {
	currentConfigApply = make(map[string]*instances.File)
	skippedFiles = make(map[string]struct{})

	configDownloader := client.NewHttpConfigDownloader()

	filesMetaData, err := configDownloader.GetFilesMetadata(filesUrl, tenantID)
	if err != nil {
		slog.Error("Error getting files metadata", "filesUrl", filesUrl, "error", err)
		return nil, nil, err
	}

filesLoop:
	for _, fileData := range filesMetaData.Files {
		if fileData.Path != "" && !strings.HasSuffix(fileData.Path, "/") {
			if lastConfigApply != nil && len(lastConfigApply) > 0 {
				fileOnSystem, ok := lastConfigApply[fileData.Path]
				if ok && !fileData.LastModified.AsTime().After(fileOnSystem.LastModified.AsTime()) {
					slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.Path)
					currentConfigApply[fileData.Path] = lastConfigApply[fileData.Path]
					skippedFiles[fileData.Path] = struct{}{}
					continue filesLoop
				}

			}

			fileDownloadResponse, err := configDownloader.GetFile(fileData, filesUrl, tenantID)
			if err != nil {
				slog.Error("Error getting file data", "filesUrl", filesUrl, "err", err)
			}

			err = WriteFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
			if err != nil {
				slog.Error("Error writing to file", "filesUrl", filesUrl, "err", err)
			}

			currentConfigApply[fileData.Path] = &instances.File{
				Version:      fileData.Version,
				Path:         fileData.Path,
				LastModified: fileData.LastModified,
			}
		}
	}

	return currentConfigApply, skippedFiles, err
}
