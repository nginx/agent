/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
	"github.com/nginx/agent/v3/internal/datasource/os"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_config_writer.go . ConfigWriterInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/config mock_config_writer.go | sed -e s\\/config\\\\.\\/\\/g > mock_config_writer_fixed.go"
//go:generate mv mock_config_writer_fixed.go mock_config_writer.go

type ConfigWriterInterface interface {
	Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error)
	isFilePathValid(filePath string) bool
	doesFileRequireUpdate(previousFileCache os.FileCache, fileData *instances.File) (latest bool)
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
)

func NewConfigWriter(configWriterParameters *ConfigWriterParameters) *ConfigWriter {
	if configWriterParameters == nil {
		configWriterParameters.configClient = client.NewHttpConfigClient(configWriterParameters.Client.Timeout)
	}

	return &ConfigWriter{
		configClient: configWriterParameters.configClient,
	}
}

func (cw *ConfigWriter) Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error) {
	currentFileCache = os.FileCache{}
	skippedFiles = make(map[string]struct{})

	filesMetaData, err := cw.configClient.GetFilesMetadata(filesUrl, tenantID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("error getting files metadata from %s: %w", filesUrl, err)
	}

	for _, fileData := range filesMetaData.Files {
		if isFilePathValid(fileData.Path) {
			if !doesFileRequireUpdate(previousFileCache, fileData) {
				slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.Path)
				currentFileCache[fileData.Path] = previousFileCache[fileData.Path]
				skippedFiles[fileData.Path] = struct{}{}
				continue
			}

			fileDownloadResponse, err := cw.configClient.GetFile(fileData, filesUrl, tenantID.String())
			if err != nil {
				return nil, nil, fmt.Errorf("error getting file data from %s: %w", filesUrl, err)
			}

			err = os.WriteFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
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

func isFilePathValid(filePath string) bool {
	return filePath != "" && !strings.HasSuffix(filePath, "/")
}

func doesFileRequireUpdate(previousFileCache os.FileCache, fileData *instances.File) (latest bool) {
	if previousFileCache != nil && len(previousFileCache) > 0 {
		fileOnSystem, ok := previousFileCache[fileData.Path]
		return ok && fileOnSystem.LastModified.AsTime().Before(fileData.LastModified.AsTime())
	}
	return false
}
