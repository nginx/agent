package datasource

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
	"github.com/nginx/agent/v3/internal/datasource/os"
)

type (
	ConfigWriterParameters struct {
		configDownloader client.HttpConfigClientInterface
	}

	ConfigWriter struct {
		configDownloader client.HttpConfigClientInterface
	}
)

func NewConfigWriter(configWriterParameters *ConfigWriterParameters) *ConfigWriter {
	if configWriterParameters == nil {
		configWriterParameters.configDownloader = client.NewHttpConfigClient()
	}

	return &ConfigWriter{
		configDownloader: configWriterParameters.configDownloader,
	}
}

func (cw *ConfigWriter) Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error) {
	currentFileCache = os.FileCache{}
	skippedFiles = make(map[string]struct{})

	filesMetaData, err := cw.configDownloader.GetFilesMetadata(filesUrl, tenantID)
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

			fileDownloadResponse, err := cw.configDownloader.GetFile(fileData, filesUrl, tenantID)
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
		return !(ok && !fileData.LastModified.AsTime().After(fileOnSystem.LastModified.AsTime()))
	}
	return true
}
