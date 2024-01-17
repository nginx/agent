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
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_http_config_downloader.go . HttpConfigDownloaderInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/client mock_http_config_downloader.go | sed -e s\\/client\\\\.\\/\\/g > mock_http_config_downloader_fixed.go"
//go:generate mv mock_http_config_downloader_fixed.go mock_http_config_downloader.go
type HttpConfigClientInterface interface {
	GetFilesMetadata(filesUrl string, tenantID uuid.UUID) (*instances.Files, error)
	GetFile(file *instances.File, filesUrl string, tenantID uuid.UUID) (*instances.FileDownloadResponse, error)
}

type HttpConfigClient struct {
	httpClient http.Client
}

func NewHttpConfigClient() *HttpConfigClient {
	httpClient := http.Client{
		Timeout: time.Second * 10,
	}

	return &HttpConfigClient{
		httpClient: httpClient,
	}
}

func (hcd *HttpConfigClient) GetFilesMetadata(filesUrl string, tenantID uuid.UUID) (*instances.Files, error) {
	files := instances.Files{}

	req, err := http.NewRequest(http.MethodGet, filesUrl, nil)
	req.Header.Set("tenantId", tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("error creating GetFilesMetadata request, filesUrl:%v, error: %v", filesUrl, err)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFilesMetadata request, filesUrl:%v, error: %v", filesUrl, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFilesMetadata response body, filesUrl:%v, error: %v", filesUrl, err)
	}

	// version is an unknown field
	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	err = pb.Unmarshal(data, &files)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling GetFilesMetadata response, responseData:%v, error: %v", data, err)
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
	req.Header.Set("tenantId", tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("error creating GetFile request, filesUrl:%v, error: %v", filesUrl, err)
	}

	resp, err := hcd.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making GetFile request, filesUrl:%v, error: %v", filesUrl, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading GetFile response body, filesUrl:%v, error: %v", filesUrl, err)
	}

	// type is an unknown field
	pb := protojson.UnmarshalOptions{DiscardUnknown: true}
	err = pb.Unmarshal(data, &response)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling GetFile response, responseData:%v, error: %v", data, err)
	}

	return &response, err
}
