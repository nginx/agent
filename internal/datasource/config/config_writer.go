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

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/client"
)

const (
	filePermissions = 0o600
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ConfigWriterInterface

type (
	ConfigWriterInterface interface {
		Write(ctx context.Context, request *v1.ManagementPlaneRequest_ConfigApplyRequest,
			instanceID string) (skippedFiles CacheContent, err error)
		Complete(ctx context.Context) (err error)
		SetConfigClient(configClient client.ConfigClient)
		Rollback(ctx context.Context, skippedFiles CacheContent,
			request *v1.ManagementPlaneRequest_ConfigApplyRequest, instanceID string) error
	}

	ConfigWriter struct {
		configClient       client.ConfigClient
		allowedDirectories []string
		fileCache          FileCacheInterface
		currentFileCache   CacheContent
	}
)

func NewConfigWriter(agentConfig *config.Config, fileCache FileCacheInterface,
	configClient client.ConfigClient,
) (*ConfigWriter, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("failed to create config writer, agent config is nil")
	}

	return &ConfigWriter{
		configClient:       configClient,
		allowedDirectories: agentConfig.AllowedDirectories,
		fileCache:          fileCache,
	}, nil
}

func (cw *ConfigWriter) SetConfigClient(configClient client.ConfigClient) {
	cw.configClient = configClient
}

func (cw *ConfigWriter) Rollback(ctx context.Context, skippedFiles CacheContent,
	_ *v1.ManagementPlaneRequest_ConfigApplyRequest, instanceID string,
) error {
	// The below will be done in a followup PR
	// if the config apply has failed before a file is written the skipped files will be empty
	// if one but not all the files failed to download should handle comparing the hash in doesFileRequireUpdate
	slog.DebugContext(ctx, "Rolling back NGINX config changes due to error")
	if cw.fileCache.CacheContent() == nil {
		return fmt.Errorf("error rolling back, no instance file cache found for instance %s", instanceID)
	}
	for key, fileMeta := range cw.fileCache.CacheContent() {
		if _, ok := skippedFiles[key]; ok {
			continue
		}

		err := cw.updateFile(ctx, fileMeta)
		if err != nil {
			return err
		}
	}

	return nil
}

// has cognitive-complexity of 11 due to the number or nil and error checks
// nolint: revive
func (cw *ConfigWriter) Write(ctx context.Context,
	request *v1.ManagementPlaneRequest_ConfigApplyRequest, instanceID string,
) (skippedFiles CacheContent, err error) {
	slog.DebugContext(ctx, "Write nginx config")
	currentFileCache, skippedFiles := make(CacheContent), make(CacheContent)

	cacheContent, err := cw.fileCache.ReadFileCache(ctx)
	if err != nil {
		slog.Warn("Unable to read file cache")
	}

	filesOverview := request.ConfigApplyRequest.GetOverview()

	if filesOverview == nil {
		filesOverview, err = cw.getFileOverview(ctx, request, instanceID)
		if err != nil {
			return nil, err
		}
	}

	for _, fileData := range filesOverview.GetFiles() {
		if cacheContent != nil && !doesFileRequireUpdate(cacheContent, fileData.GetFileMeta()) {
			slog.DebugContext(
				ctx,
				"Skipping file as latest version is already on disk",
				"file_path", fileData.GetFileMeta().GetName(),
			)
			currentFileCache[fileData.GetFileMeta().GetName()] = cacheContent[fileData.GetFileMeta().GetName()]
			skippedFiles[fileData.GetFileMeta().GetName()] = fileData.GetFileMeta()

			continue
		}
		slog.DebugContext(
			ctx,
			"Updating file, latest version not on disk",
			"file_path", fileData.GetFileMeta().GetName(),
		)
		if updateErr := cw.updateFile(ctx, fileData.GetFileMeta()); updateErr != nil {
			return nil, updateErr
		}
		currentFileCache[fileData.GetFileMeta().GetName()] = fileData.GetFileMeta()
	}

	cw.currentFileCache = currentFileCache

	if cacheContent != nil {
		err = cw.removeFiles(ctx, currentFileCache, cacheContent)
	}

	return skippedFiles, err
}

func (cw *ConfigWriter) removeFiles(ctx context.Context, currentFileCache, fileCache CacheContent) error {
	for _, file := range fileCache {
		if _, ok := currentFileCache[file.GetName()]; !ok {
			slog.DebugContext(ctx, "Removing file", "file_path", file.GetName())
			if err := os.Remove(file.GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error removing file: %s error: %w", file.GetName(), err)
			}
		}
	}

	return nil
}

func (cw *ConfigWriter) getFileOverview(ctx context.Context, request *v1.ManagementPlaneRequest_ConfigApplyRequest,
	instanceID string,
) (fileOverview *v1.FileOverview, err error) {
	fileOverview, err = cw.configClient.GetOverview(ctx, &v1.GetOverviewRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: logger.GetCorrelationID(ctx),
			Timestamp:     timestamppb.Now(),
		},
		ConfigVersion: request.ConfigApplyRequest.GetConfigVersion(),
	})
	if err != nil {
		return nil, fmt.Errorf("error getting files metadata: %w", err)
	}

	if len(fileOverview.GetFiles()) == 0 {
		slog.DebugContext(ctx, "No file metadata for instance", "instance_id", instanceID)
		return nil, fmt.Errorf("error getting files metadata, no metadata exists for instance: %s", instanceID)
	}

	return fileOverview, nil
}

func (cw *ConfigWriter) updateFile(ctx context.Context, fileData *v1.FileMeta,
) error {
	if !cw.isFilePathValid(ctx, fileData.GetName()) {
		return fmt.Errorf("invalid file path: %s", fileData.GetName())
	}
	fileDownloadResponse, fetchErr := cw.configClient.GetFile(ctx, &v1.GetFileRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: logger.GetCorrelationID(ctx),
			Timestamp:     timestamppb.Now(),
		},
		FileMeta: fileData,
	})
	if fetchErr != nil {
		return fmt.Errorf("error getting file data for %s: %w", fileData.GetName(), fetchErr)
	}

	fetchErr = writeFile(ctx, fileDownloadResponse.GetContents(), fileData.GetName())

	if fetchErr != nil {
		return fmt.Errorf("error writing to file %s: %w", fileData.GetName(), fetchErr)
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

func doesFileRequireUpdate(fileCache CacheContent, fileData *v1.FileMeta) (updateRequired bool) {
	if len(fileCache) > 0 {
		fileOnSystem, ok := fileCache[fileData.GetName()]
		if !ok {
			return true
		}

		// TODO: Use hash
		return ok && fileOnSystem.GetModifiedTime().AsTime().Before(fileData.GetModifiedTime().AsTime())
	}

	return false
}
