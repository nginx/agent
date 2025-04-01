// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func FileMeta(fileName, fileHash string) *mpi.FileMeta {
	lastModified, _ := CreateProtoTime("2024-01-09T13:22:21Z")

	return &mpi.FileMeta{
		ModifiedTime: lastModified,
		Name:         fileName,
		Hash:         fileHash,
		Permissions:  "0600",
	}
}

func CertMeta(fileName, fileHash string) *mpi.FileMeta {
	lastModified, _ := CreateProtoTime("2024-01-09T13:22:21Z")

	return &mpi.FileMeta{
		Name:         fileName,
		Hash:         fileHash,
		ModifiedTime: lastModified,
		Permissions:  "0600",
		FileType: &mpi.FileMeta_CertificateMeta{
			CertificateMeta: &mpi.CertificateMeta{
				SerialNumber: "12345-67890",
				Issuer: &mpi.X509Name{
					Country:            []string{"IE"},
					Organization:       []string{"F5"},
					OrganizationalUnit: []string{"NGINX"},
					Locality:           []string{"Cork"},
					Province:           []string{"Munster"},
					StreetAddress:      []string{"90 South Mall"},
					PostalCode:         []string{"T12 KXV9"},
					CommonName:         "Example Name",
					Names: []*mpi.AttributeTypeAndValue{
						{Type: "email", Value: "noreply@nginx.com"},
						{Type: "phone", Value: "+353217355700"},
					},
					ExtraNames: []*mpi.AttributeTypeAndValue{
						{Type: "customID", Value: "98765"},
					},
				},

				SignatureAlgorithm: mpi.SignatureAlgorithm_SIGNATURE_ALGORITHM_UNKNOWN,
				PublicKeyAlgorithm: "",
			},
		},
	}
}

func FileOverview(filePath, fileHash string) *mpi.FileOverview {
	return &mpi.FileOverview{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         filePath,
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
		},
		ConfigVersion: CreateConfigVersion(),
	}
}
