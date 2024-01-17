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

func (cw *ConfigWriter) Write(lastConfigApply map[string]*instances.File, filesUrl string, tenantID uuid.UUID) (currentConfigApply map[string]*instances.File, skippedFiles map[string]struct{}, err error) {
	currentConfigApply = make(map[string]*instances.File)
	skippedFiles = make(map[string]struct{})

	filesMetaData, err := cw.configDownloader.GetFilesMetadata(filesUrl, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting files metadata, filesUrl: %v, error: %v", filesUrl, err)
	}

filesLoop:
	for _, fileData := range filesMetaData.Files {
		if isFilePathValid(fileData.Path) {
			if !doesFileRequireUpdate(lastConfigApply, fileData) {
				slog.Debug("Skipping file as latest version is already on disk", "filePath", fileData.Path)
				currentConfigApply[fileData.Path] = lastConfigApply[fileData.Path]
				skippedFiles[fileData.Path] = struct{}{}
				continue filesLoop
			}

			fileDownloadResponse, err := cw.configDownloader.GetFile(fileData, filesUrl, tenantID)
			if err != nil {
				return nil, nil, fmt.Errorf("error getting file data, filesUrl:%v, error: %v", filesUrl, err)
			}

			err = os.WriteFile(fileDownloadResponse.FileContent, fileDownloadResponse.FilePath)
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

func isFilePathValid(filePath string) bool {
	if filePath != "" && !strings.HasSuffix(filePath, "/") {
		return true
	}
	return false
}

func doesFileRequireUpdate(lastConfigApply map[string]*instances.File, fileData *instances.File) (latest bool) {
	if lastConfigApply != nil && len(lastConfigApply) > 0 {
		fileOnSystem, ok := lastConfigApply[fileData.Path]
		if ok && !fileData.LastModified.AsTime().After(fileOnSystem.LastModified.AsTime()) {
			return false
		}
		return true
	}
	return true
}
