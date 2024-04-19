// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateManagementPlaneRequestConfigApplyRequest() *v1.ManagementPlaneRequest_ConfigApplyRequest {
	return &v1.ManagementPlaneRequest_ConfigApplyRequest{
		ConfigApplyRequest: &v1.ConfigApplyRequest{
			ConfigVersion: &v1.ConfigVersion{
				Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
				InstanceId: protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId(),
			},
		},
	}
}

func CreateGetOverviewRequest() *v1.GetOverviewRequest {
	return &v1.GetOverviewRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     "",
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		ConfigVersion: &v1.ConfigVersion{
			Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
			InstanceId: protos.GetNginxOssInstance().GetInstanceMeta().GetInstanceId(),
		},
	}
}

func CreateGetFileRequest() (*v1.GetFileRequest, error) {
	lastModified, err := protos.CreateProtoTime("2024-01-09T13:22:21Z")
	return &v1.GetFileRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     "",
			CorrelationId: "dfsbhj6-bc92-30c1-a9c9-85591422068e",
			Timestamp:     timestamppb.Now(),
		},
		FileMeta: &v1.FileMeta{
			ModifiedTime: lastModified,
			Name:         "nginx.conf",
			Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
	}, err
}

func CreateGetOverviewResponse() (*v1.GetOverviewResponse, error) {
	lastModified, err := protos.CreateProtoTime("2024-01-09T13:22:21Z")
	return &v1.GetOverviewResponse{
		Overview: &v1.FileOverview{
			Files: []*v1.File{
				{
					FileMeta: &v1.FileMeta{
						ModifiedTime: lastModified,
						Name:         "nginx.conf",
						Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
					},
				},
			},
		},
	}, err
}

func CreateGetFileResponse() *v1.GetFileResponse {
	content := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	return &v1.GetFileResponse{
		Contents: &v1.FileContents{
			Contents: content,
		},
	}
}
