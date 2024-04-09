// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

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

func (hcd *HTTPConfigClient) GetFilesMetadata(ctx context.Context, request *v1.GetOverviewRequest) (*v1.FileOverview,
	error,
) {
	return nil, fmt.Errorf("not implemented")
}

func (hcd *HTTPConfigClient) GetFile(ctx context.Context, request *v1.GetFileRequest) (*v1.FileContents, error) {
	return nil, fmt.Errorf("not implemented")
}
