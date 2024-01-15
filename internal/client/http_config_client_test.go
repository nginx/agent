/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateTestIds() (uuid.UUID, uuid.UUID, error) {
	tenantId, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	if err != nil {
		fmt.Printf("Error creating tenantId: %v", err)
	}

	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	if err != nil {
		fmt.Printf("Error creating instanceId: %v", err)
	}

	return tenantId, instanceId, err
}

func TestGetFilesMetadata(t *testing.T) {
	tenantId, instanceId, err := CreateTestIds()

	timeFile1, err := time.Parse(time.RFC3339, "2024-01-08T13:22:25Z")
	protoTimeFile1 := timestamppb.New(timeFile1)

	timeFile2, err := time.Parse(time.RFC3339, "2024-01-08T13:22:21Z")
	protoTimeFile2 := timestamppb.New(timeFile2)

	testDataResponse := &instances.Files{
		Files: []*instances.File{
			{
				LastModified: protoTimeFile1,
				Path:         "/usr/local/etc/nginx/locations/test.conf",
				Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
			},
			{
				LastModified: protoTimeFile2,
				Path:         "/usr/local/etc/nginx/nginx.conf",
				Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
			},
		},
	}

	test := "{\"files\":[{\"lastModified\":\"2024-01-08T13:22:25Z\",\"path\":\"/usr/local/etc/nginx/locations/test.conf\",\"version\":\"Rh3phZuCRwNGANTkdst51he_0WKWy.tZ\"},{\"lastModified\":\"2024-01-08T13:22:21Z\",\"path\":\"/usr/local/etc/nginx/nginx.conf\",\"version\":\"BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c\"}],\"instanceId\":\"aecea348-62c1-4e3d-b848-6d6cdeb1cb9c\",\"type\":\"\"}\n"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesUrl := fmt.Sprintf("%v/instance/%s/files/", ts.URL, instanceId)

	hcd := NewHttpConfigDownloader()

	resp, err := hcd.GetFilesMetadata(filesUrl, tenantId)
	if err != nil {
		log.Fatal(err)
	}

	assert.NoError(t, err)
	assert.Equal(t, resp.String(), testDataResponse.String())
}

func TestGetFile(t *testing.T) {
	tenantId, instanceId, err := CreateTestIds()
	if err != nil {
		fmt.Printf("Failed to create ID's")
		log.Fatal(err)
	}

	test := "{\"encoded\":true,\"fileContent\":\"bG9jYXRpb24gL3Rlc3QgewogICAgcmV0dXJuIDIwMCAiVGVzdCBsb2NhdGlvblxuIjsKfQ==\",\"filePath\":\"/usr/local/etc/nginx/locations/test.conf\",\"instanceId\":\"aecea348-62c1-4e3d-b848-6d6cdeb1cb9c\",\"type\":\"\"}\n"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, test)
	}))
	defer ts.Close()

	filesUrl := fmt.Sprintf("%v/instance/%s/files/", ts.URL, instanceId)

	time, err := time.Parse(time.RFC3339, "2024-01-08T13:22:25Z")
	protoTime := timestamppb.New(time)

	file := instances.File{
		LastModified: protoTime,
		Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		Path:         "/usr/local/etc/nginx/locations/test.conf",
	}

	testDataResponse := &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    "/usr/local/etc/nginx/locations/test.conf",
		InstanceId:  instanceId.String(),
		FileContent: []byte("location /test {\n    return 200 \"Test location\\n\";\n}"),
	}

	hcd := NewHttpConfigDownloader()
	resp, err := hcd.GetFile(&file, filesUrl, tenantId)
	if err != nil {
		log.Fatal(err)
	}

	assert.NoError(t, err)
	assert.Equal(t, resp.String(), testDataResponse.String())
}
