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
	"github.com/nginx/agent/v3/internal/datasource/nginx"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_config_writer.go . ConfigWriterInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/config mock_config_writer.go | sed -e s\\/config\\\\.\\/\\/g > mock_config_writer_fixed.go"
//go:generate mv mock_config_writer_fixed.go mock_config_writer.go
type ConfigWriterInterface interface {
	Write(filesUrl string, tenantID uuid.UUID) (err error)
	Complete() (err error)
}

const (
	cacheLocation = "/var/lib/nginx-agent/config/%v/cache.json"
)

type (
	Client struct {
		Timeout time.Duration
	}

	ConfigWriterParameters struct {
		configClient client.HttpConfigClientInterface
		Client       Client
		cachePath string
	}

	ConfigWriter struct {
		configClient       client.HttpConfigClientInterface
		previouseFileCache FileCache
		currentFileCache   FileCache
		cachePath          string
		dataplaneConfig    nginx.DataplaneConfigInterface
	}

	// map of files with filepath as key
	FileCache = map[string]*instances.File
)

func NewConfigWriter(configWriterParameters *ConfigWriterParameters, instanceId string) *ConfigWriter {

	if configWriterParameters.cachePath == "" {
		configWriterParameters.cachePath = fmt.Sprintf(cacheLocation, instanceId)
	}
	

	if configWriterParameters.configClient == nil {
		configWriterParameters.configClient = client.NewHttpConfigClient(configWriterParameters.Client.Timeout)
	}

	previouseFileCache, err := readInstanceCache(configWriterParameters.cachePath)
	if err != nil {
		slog.Info("Failed to Read cache %s ", configWriterParameters.cachePath, "err", err)
	}

	return &ConfigWriter{
		configClient:       configWriterParameters.configClient,
		previouseFileCache: previouseFileCache,
		cachePath:          configWriterParameters.cachePath,
	}
}

func (cw *ConfigWriter) Write(filesUrl string, tenantID uuid.UUID) (err error) {
	currentFileCache := FileCache{}
	skippedFiles := make(map[string]struct{})

	filesMetaData, err := cw.configClient.GetFilesMetadata(filesUrl, tenantID.String())
	if err != nil {
		return fmt.Errorf("error getting files metadata from %s: %w", filesUrl, err)
	}

	for _, fileData := range filesMetaData.Files {
		if isFilePathValid(fileData.Path) {
			if !doesFileRequireUpdate(cw.previouseFileCache, fileData) {
				slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.Path)
				currentFileCache[fileData.Path] = cw.previouseFileCache[fileData.Path]
				skippedFiles[fileData.Path] = struct{}{}
				continue
			}

			fileDownloadResponse, err := cw.configClient.GetFile(fileData, filesUrl, tenantID.String())
			if err != nil {
				return fmt.Errorf("error getting file data from %s: %w", filesUrl, err)
			}

			err = writeFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
			if err != nil {
				return fmt.Errorf("error writing to file %s: %w", fileDownloadResponse.FilePath, err)
			}

			currentFileCache[fileData.Path] = &instances.File{
				Version:      fileData.Version,
				Path:         fileData.Path,
				LastModified: fileData.LastModified,
			}
		}
	}

	cw.currentFileCache = currentFileCache

	return err
}

func (cw *ConfigWriter) Complete() error {
	cache, err := json.MarshalIndent(cw.currentFileCache, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling cache data from %s: %w", cw.cachePath, err)
	}

	err = writeFile(cache, cw.cachePath)
	if err != nil {
		return fmt.Errorf("error writing cache to %s: %w", cw.cachePath, err)
	}

	return err
}

func (cw *ConfigWriter) SetDataplaneConfig(dataplaneConfig nginx.DataplaneConfigInterface) error {
	cw.dataplaneConfig = dataplaneConfig
	return nil
}

func (cw *ConfigWriter) Reload() error {
	return cw.dataplaneConfig.Reload()
}

func (cw *ConfigWriter) Validate() error {
	return cw.dataplaneConfig.Validate()
}

func writeFile(fileContent []byte, filePath string) error {
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

func readInstanceCache(cachePath string) (previousFileCache FileCache, err error) {
	previousFileCache = FileCache{}

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

func isFilePathValid(filePath string) (validPath bool) {
	return filePath != "" && !strings.HasSuffix(filePath, "/")
}

func doesFileRequireUpdate(previousFileCache FileCache, fileData *instances.File) (updateRequired bool) {
	if len(previousFileCache) > 0 {
		fileOnSystem, ok := previousFileCache[fileData.Path]
		return ok && fileOnSystem.LastModified.AsTime().Before(fileData.LastModified.AsTime())
	}
	return false
}
