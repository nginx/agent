// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	helpers "github.com/nginx/agent/v3/test"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"
)

func TestGetFilesMetadata(t *testing.T) {
	ctx := context.TODO()
	tenantID, instanceID := helpers.CreateTestIDs(t)

	fileTime1, err := helpers.CreateProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	fileTime2, err := helpers.CreateProtoTime("2024-01-08T13:22:21Z")
	require.NoError(t, err)

	testDataResponse := &instances.Files{
		Files: []*instances.File{
			{
				LastModified: fileTime1,
				Path:         "/usr/local/etc/nginx/locations/test.conf",
				Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
			},
			{
				LastModified: fileTime2,
				Path:         "/usr/local/etc/nginx/nginx.conf",
				Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
			},
		},
	}

	test := `{
		"files":[
			{
				"lastModified":"2024-01-08T13:22:25Z",
				"path":"/usr/local/etc/nginx/locations/test.conf",
				"version":"Rh3phZuCRwNGANTkdst51he_0WKWy.tZ"
			},
			{
				"lastModified":"2024-01-08T13:22:21Z",
				"path":"/usr/local/etc/nginx/nginx.conf",
				"version":"BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c"
			}
		],
		"instanceID":"aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		"type":""
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesURL := fmt.Sprintf("%s/instances/%s/files/", ts.URL, instanceID.String())

	hcd := NewHTTPConfigClient(time.Second * 10)

	resp, err := hcd.GetFilesMetadata(ctx, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)
	assert.Equal(t, resp.String(), testDataResponse.String())
}

func TestGetFile(t *testing.T) {
	ctx := context.TODO()
	tenantID, instanceID := helpers.CreateTestIDs(t)

	test := `{
		"encoded":true,
		"fileContent":"bG9jYXRpb24gL3Rlc3QgewogICAgcmV0dXJuIDIwMCAiVGVzdCBsb2NhdGlvblxuIjsKfQ==",
		"filePath":"/usr/local/etc/nginx/locations/test.conf",
		"instanceID":"aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		"type":""
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesURL := fmt.Sprintf("%s/instances/%s/files/", ts.URL, instanceID.String())

	fileTime, err := helpers.CreateProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	file := instances.File{
		LastModified: fileTime,
		Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		Path:         "/usr/local/etc/nginx/locations/test.conf",
	}

	testDataResponse := &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    "/usr/local/etc/nginx/locations/test.conf",
		InstanceId:  instanceID.String(),
		FileContent: []byte("location /test {\n    return 200 \"Test location\\n\";\n}"),
	}

	hcd := NewHTTPConfigClient(time.Second * 10)

	resp, err := hcd.GetFile(ctx, &file, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)
	assert.Equal(t, testDataResponse.String(), resp.String())
}
