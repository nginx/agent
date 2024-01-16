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

type (
	FileSourceParameters struct {
		configDownloader client.HttpConfigDownloaderInterface
	}

	// TODO: Naming of this ?
	FileSource struct {
		configDownloader client.HttpConfigDownloaderInterface
	}
)

func NewFileSource(fileSourceParameters *FileSourceParameters) *FileSource {
	if fileSourceParameters == nil {
		fileSourceParameters.configDownloader = client.NewHttpConfigDownloader()
	}

	return &FileSource{
		configDownloader: fileSourceParameters.configDownloader,
	}
}

func WriteFile(fileContent []byte, filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		slog.Debug("File does not exist, creating")
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

func ReadCache(cachePath string) (map[string]*instances.File, error) {
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

func (fs *FileSource) UpdateInstanceConfig(lastConfigApply map[string]*instances.File, filesUrl string, tenantID uuid.UUID) (currentConfigApply map[string]*instances.File, skippedFiles map[string]struct{}, err error) {
	currentConfigApply = make(map[string]*instances.File)
	skippedFiles = make(map[string]struct{})

	filesMetaData, err := fs.configDownloader.GetFilesMetadata(filesUrl, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting files metadata, filesUrl: %v, error: %v", filesUrl, err)
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

			fileDownloadResponse, err := fs.configDownloader.GetFile(fileData, filesUrl, tenantID)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting file data, filesUrl:%v, error: %v", filesUrl, err)
			}

			err = WriteFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
			if err != nil {
				return nil, nil, fmt.Errorf("error writing to file, filePath:%v, error: %v", fileDownloadResponse.FilePath, err)
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
