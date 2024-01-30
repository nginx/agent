/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate . HttpConfigClientInterface
const tenantHeader = "tenantId"

type HttpConfigClientInterface interface {
	GetFilesMetadata(filesUrl string, tenantID uuid.UUID) (*instances.Files, error)
	GetFile(file *instances.File, filesUrl string, tenantID uuid.UUID) (*instances.FileDownloadResponse, error)
}

type HttpConfigClient struct {
	httpClient http.Client
}

func NewHttpConfigClient(timeout time.Duration) *HttpConfigClient {
	httpClient := http.Client{
		Timeout: timeout,
	}

	return &HttpConfigClient{
		httpClient: httpClient,
	}
}

func (hcd *HttpConfigClient) GetFilesMetadata(filesUrl string, tenantID uuid.UUID) (*instances.Files, error) {
	files := instances.Files{}

	req, err := http.NewRequest(http.MethodGet, filesUrl, nil)
	req.Header.Set(tenantHeader, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("error creating GetFilesMetadata request %s: %w", filesUrl, err)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFilesMetadata request %s: %w", filesUrl, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFilesMetadata response body from %s: %w", filesUrl, err)
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

func (hcd *HttpConfigClient) GetFile(file *instances.File, filesUrl string, tenantID uuid.UUID) (*instances.FileDownloadResponse, error) {
	response := instances.FileDownloadResponse{}
	params := url.Values{}

	params.Add("version", file.Version)
	params.Add("encoded", "true")

	filePath := url.QueryEscape(file.Path)

	fileUrl := fmt.Sprintf("%v%v?%v", filesUrl, filePath, params.Encode())

	req, err := http.NewRequest(http.MethodGet, fileUrl, nil)
	req.Header.Set(tenantHeader, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("error creating GetFile request %s: %w", filesUrl, err)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFile request %s: %w", filesUrl, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFile response body from %s: %w", filesUrl, err)
	}

	// type is returned for the rest api but is not in the proto definitions so needs to be discarded
	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	err = pb.Unmarshal(data, &response)

	if err != nil {
		slog.Debug("Error unmarshalling GetFile Response", "data", string(data))
		return nil, fmt.Errorf("error unmarshalling GetFile response: %w", err)
	}

	return &response, err
}
