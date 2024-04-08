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

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestGetFilesMetadata(t *testing.T) {
	ctx := context.Background()
	tenantID, instanceID := helpers.CreateTestIDs(t)

	fileTime1, err := protos.CreateProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	fileTime2, err := protos.CreateProtoTime("2024-01-08T13:22:21Z")
	require.NoError(t, err)

	testDataResponse := &v1.FileOverview{
		Files: []*v1.File{
			{
				FileMeta: &v1.FileMeta{
					ModifiedTime: fileTime1,
					Name:         "/usr/local/etc/nginx/locations/test.conf",
					Hash:         "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
				},
			},
			{
				FileMeta: &v1.FileMeta{
					ModifiedTime: fileTime2,
					Name:         "/usr/local/etc/nginx/nginx.conf",
					Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
				},
			},
		},
		ConfigVersion: &v1.ConfigVersion{
			Version:    "2",
			InstanceId: "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		},
	}

	test := `{
		"files":[
			{
				"file_meta":
					{
						"name":"/usr/local/etc/nginx/locations/test.conf",
						"hash":"Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
						"modified_time":"2024-01-08T13:22:25Z"
					}
			},
			{
				"file_meta":
					{
						"name":"/usr/local/etc/nginx/nginx.conf",
						"hash":"BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
						"modified_time":"2024-01-08T13:22:21Z"
					}
			}
		],
		"config_version":
			{
				"instance_id":"aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
				"version":"2"
			}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesURL := fmt.Sprintf("%s/instances/%s/files/", ts.URL, instanceID.String())

	hcd := NewHTTPConfigClient(time.Second * 10)

	resp, err := hcd.GetFilesMetadata(ctx, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)
	assert.Equal(t, testDataResponse.String(), resp.String())
}

func TestGetFile(t *testing.T) {
	ctx := context.Background()
	tenantID, instanceID := helpers.CreateTestIDs(t)

	test := `{"contents":"bG9jYXRpb24gL3Rlc3QgewogICAgcmV0dXJuIDIwMCAiVGVzdCBsb2NhdGlvblxuIjsKfQ=="}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesURL := fmt.Sprintf("%s/instances/%s/files/", ts.URL, instanceID.String())

	fileTime, err := protos.CreateProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	file := v1.FileMeta{
		ModifiedTime: fileTime,
		Hash:         "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		Name:         "/usr/local/etc/nginx/locations/test.conf",
	}

	testDataResponse := &v1.FileContents{
		Contents: []byte("location /test {\n    return 200 \"Test location\\n\";\n}"),
	}

	hcd := NewHTTPConfigClient(time.Second * 10)

	resp, err := hcd.GetFile(ctx, &file, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)
	assert.Equal(t, testDataResponse.String(), resp.String())
}

func TestGetFilesMetadata_ErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		filesURL   string
		tenantID   string
		instanceID string
	}{
		{
			name:       "Test 1: Provide empty parameters",
			filesURL:   "",
			tenantID:   "",
			instanceID: "",
		},
		{
			name:       "Test 2: Provide invalid URL",
			filesURL:   "::/\\",
			tenantID:   "",
			instanceID: "",
		},
	}
	hcd := NewHTTPConfigClient(time.Second * 10)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := hcd.GetFilesMetadata(
				context.Background(), test.filesURL,
				test.tenantID, test.instanceID,
			)
			require.Error(t, err)
			assert.Empty(t, f)
		})
	}
}
