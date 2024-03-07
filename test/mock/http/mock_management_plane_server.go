// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package http

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
)

type ManagementServer struct {
	configDirectory string
	server          *gin.Engine
}

func NewManagementServer(configDirectory string) *ManagementServer {
	ms := &ManagementServer{
		configDirectory: configDirectory,
	}

	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	logger := slog.New(handler)

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	server.UseRawPath = true
	server.Use(sloggin.NewWithConfig(logger, sloggin.Config{DefaultLevel: slog.LevelDebug}))
	RegisterHandlersWithOptions(server, ms, GinServerOptions{BaseURL: "/api/v1"})

	ms.server = server

	return ms
}

// nolint: revive
// Return all files metadata for a data plane instance.
// (GET /instances/{instanceID}/files/)
func (ms *ManagementServer) GetInstanceFilesMetadata(
	ctx *gin.Context,
	instanceID string,
	params GetInstanceFilesMetadataParams,
) {
	files, err := ms.getFiles()

	if err != nil {
		slog.Error("Unable to get files", "error", err)
		ctx.JSON(http.StatusInternalServerError, err)
	} else {
		response := &FilesResponse{
			InstanceId: instanceID,
			Files:      files,
		}

		ctx.JSON(http.StatusOK, response)
	}
}

// nolint: revive
// Download a specific file for a data plane instance.
// (GET /instances/{instanceID}/files/{filePath})
func (ms *ManagementServer) GetInstanceFile(
	ctx *gin.Context,
	instanceID, filePath string,
	params GetInstanceFileParams,
) {
	files, err := ms.getFiles()
	if err != nil {
		slog.Error("Unable to get files", "error", err)
		ctx.JSON(http.StatusInternalServerError, err)

		return
	}

	for _, file := range files {
		if file.Path == filePath {
			content, err := os.ReadFile(filepath.Join(ms.configDirectory, filePath))
			if err != nil {
				slog.Error("Unable to read file", "file_path", filePath, "error", err)
				ctx.JSON(http.StatusInternalServerError, err)

				return
			}

			if params.Encoded {
				response := &DownloadResponseEncoded{
					Encoded:     true,
					FileContent: content,
					FilePath:    filePath,
					InstanceId:  instanceID,
					Type:        ENCODEDDOWNLOADRESPONSE,
				}

				ctx.JSON(http.StatusOK, response)

				return
			}

			response := &DownloadResponse{
				Encoded:     false,
				FileContent: string(content),
				FilePath:    filePath,
				InstanceId:  instanceID,
				Type:        DOWNLOADRESPONSE,
			}

			ctx.JSON(http.StatusOK, response)

			return
		}
	}

	ctx.JSON(http.StatusNotFound, "")
}

func (ms *ManagementServer) StartServer(listener net.Listener) {
	slog.Info("Starting mock management plane gRPC server API", "address", listener.Addr().String())
	err := ms.server.RunListener(listener)
	if err != nil {
		slog.Error("Startup of mock management plane gRPC server API failed", "error", err)
	}
}

func (ms *ManagementServer) getFiles() ([]File, error) {
	files := []File{}

	err := filepath.Walk(ms.configDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("error", err)
			return err
		}
		if !info.IsDir() {
			files = append(files, File{
				Path:         strings.Split(path, ms.configDirectory)[1],
				Version:      "1",
				LastModified: time.Now(),
			})
		}

		return nil
	})

	return files, err
}
