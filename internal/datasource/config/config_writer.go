/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_config_writer.go . ConfigWriterInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/config mock_config_writer.go | sed -e s\\/config\\\\.\\/\\/g > mock_config_writer_fixed.go"
//go:generate mv mock_config_writer_fixed.go mock_config_writer.go
type ConfigWriterInterface interface {
	WriteFile(fileContent []byte, filePath string) error
	ReadInstanceCache(cachePath string) (FileCache, error)
	UpdateCache(currentFileCache FileCache, cachePath string) error
	isFilePathValid(filePath string) bool
	doesFileRequireUpdate(previousFileCache FileCache, fileData *instances.File) (latest bool)
	Write(previousFileCache FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache FileCache, skippedFiles map[string]struct{}, err error)
}

type (
	Client struct {
		Timeout time.Duration
	}

	ConfigWriterParameters struct {
		configClient client.HttpConfigClientInterface
		Client       Client
	}

	ConfigWriter struct {
		configClient client.HttpConfigClientInterface
	}

	// map of files with filepath as key
	FileCache = map[string]*instances.File
)

func NewConfigWriter(configWriterParameters *ConfigWriterParameters) *ConfigWriter {
	if configWriterParameters.configClient == nil {
		configWriterParameters.configClient = client.NewHttpConfigClient(configWriterParameters.Client.Timeout)
	}

	return &ConfigWriter{
		configClient: configWriterParameters.configClient,
	}
}

func (cw *ConfigWriter) Write(previousFileCache FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache FileCache, skippedFiles map[string]struct{}, err error) {
	currentFileCache = FileCache{}
	skippedFiles = make(map[string]struct{})

	filesMetaData, err := cw.configClient.GetFilesMetadata(filesUrl, tenantID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("error getting files metadata from %s: %w", filesUrl, err)
	}

	for _, fileData := range filesMetaData.Files {
		if cw.isFilePathValid(fileData.Path) {
			if !cw.doesFileRequireUpdate(previousFileCache, fileData) {
				slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.Path)
				currentFileCache[fileData.Path] = previousFileCache[fileData.Path]
				skippedFiles[fileData.Path] = struct{}{}
				continue
			}

			fileDownloadResponse, err := cw.configClient.GetFile(fileData, filesUrl, tenantID.String())
			if err != nil {
				return nil, nil, fmt.Errorf("error getting file data from %s: %w", filesUrl, err)
			}

			err = cw.WriteFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
			if err != nil {
				return nil, nil, fmt.Errorf("error writing to file %s: %w", fileDownloadResponse.FilePath, err)
			}

			currentFileCache[fileData.Path] = &instances.File{
				Version:      fileData.Version,
				Path:         fileData.Path,
				LastModified: fileData.LastModified,
			}
		}
	}

	return currentFileCache, skippedFiles, err
}

func (cw *ConfigWriter) WriteFile(fileContent []byte, filePath string) error {
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

func (cw *ConfigWriter) ReadInstanceCache(cachePath string) (FileCache, error) {
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

func (cw *ConfigWriter) UpdateCache(currentFileCache FileCache, cachePath string) error {
	cache, err := json.MarshalIndent(currentFileCache, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling cache data from %s: %w", cachePath, err)
	}

	err = cw.WriteFile(cache, cachePath)
	if err != nil {
		return fmt.Errorf("error writing cache to %s: %w", cachePath, err)
	}

	return err
}

func (cw *ConfigWriter) isFilePathValid(filePath string) bool {
	return filePath != "" && !strings.HasSuffix(filePath, "/")
}

func (cw *ConfigWriter) doesFileRequireUpdate(previousFileCache FileCache, fileData *instances.File) (latest bool) {
	if previousFileCache != nil && len(previousFileCache) > 0 {
		fileOnSystem, ok := previousFileCache[fileData.Path]
		return ok && fileOnSystem.LastModified.AsTime().Before(fileData.LastModified.AsTime())
	}
	return false
}
