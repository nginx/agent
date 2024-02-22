// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . ConfigClientInterface
const (
	tenantHeader = "tenantId"
	fileLocation = "%s/instance/%s/files/"
)

type ConfigClientInterface interface {
	GetFilesMetadata(ctx context.Context, filesURL, tenantID, instanceID string) (*instances.Files, error)
	GetFile(
		ctx context.Context,
		file *instances.File,
		filesURL, tenantID, instanceID string,
	) (*instances.FileDownloadResponse, error)
}

type HTTPConfigClient struct {
	httpClient http.Client
}

func NewHTTPConfigClient(timeout time.Duration) *HTTPConfigClient {
	httpClient := http.Client{
		Timeout: timeout,
	}

	return &HTTPConfigClient{
		httpClient: httpClient,
	}
}

func (hcd *HTTPConfigClient) GetFilesMetadata(
	ctx context.Context,
	filesURL, tenantID, instanceID string,
) (*instances.Files, error) {
	slog.Debug("Getting files metadata")
	files := instances.Files{}

	location := fmt.Sprintf(fileLocation, filesURL, instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)

	if tenantID != "" {
		req.Header.Set(tenantHeader, tenantID)
	}

	if err != nil {
		return nil, fmt.Errorf("error creating GetFilesMetadata request %s: %w", filesURL, err)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFilesMetadata request %s: %w", filesURL, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFilesMetadata response body from %s: %w", filesURL, err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	// type is returned for the rest api but is not in the proto definitions so needs to be discarded
	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	err = pb.Unmarshal(data, &files)
	if err != nil {
		slog.Debug("Error unmarshalling GetFilesMetadata Response", "data", string(data))

		return nil, fmt.Errorf("error unmarshalling GetFilesMetadata response: %w", err)
	}

	return &files, nil
}

func (hcd *HTTPConfigClient) GetFile(
	ctx context.Context,
	file *instances.File,
	filesURL, tenantID, instanceID string,
) (*instances.FileDownloadResponse, error) {
	slog.Debug("Getting file", "file_path", file.GetPath())
	response := instances.FileDownloadResponse{}
	params := url.Values{}

	params.Add("version", file.GetVersion())
	params.Add("encoded", "true")

	filePath := url.QueryEscape(file.GetPath())

	location := fmt.Sprintf(fileLocation, filesURL, instanceID)
	fileURL := fmt.Sprintf("%s%s?%s", location, filePath, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GetFile request %s: %w", filesURL, err)
	}

	if tenantID != "" {
		req.Header.Set(tenantHeader, tenantID)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFile request %s: %w", filesURL, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFile response body from %s: %w", filesURL, err)
	}

	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	// type is returned for the rest api but is not in the proto definitions so needs to be discarded
	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	err = pb.Unmarshal(data, &response)
	if err != nil {
		slog.Debug("Error unmarshalling GetFile Response", "data", string(data))

		return nil, fmt.Errorf("error unmarshalling GetFile response: %w", err)
	}

	slog.Info("response", "data", string(data))

	return &response, err
}
