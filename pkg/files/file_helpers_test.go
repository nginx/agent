// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"crypto/x509"
	_ "embed"
	"net"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/nginx/agent/v3/test/protos"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileMeta(t *testing.T) {
	tests := []struct {
		name   string
		isCert bool
	}{
		{"Test 1: conf file", false},
		{"Test 2: cert file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			var err error
			var fileMeta, expected *mpi.FileMeta
			var file *os.File

			if tt.isCert {
				_, cert := helpers.GenerateSelfSignedCert(t)

				certContents := helpers.Cert{Name: "cert.pem", Type: "CERTIFICATE", Contents: cert}
				certFile := helpers.WriteCertFiles(t, tempDir, certContents)

				require.NoError(t, err)
				expected = protos.CertMeta(certFile, "")
				fileMeta, err = FileMetaWithCertificate(certFile)
			} else {
				file = helpers.CreateFileWithErrorCheck(t, tempDir, "get_file_meta.txt")
				expected = protos.FileMeta(file.Name(), "")
				fileMeta, err = FileMeta(file.Name())
			}

			require.NoError(t, err)

			// Validate metadata
			assert.Equal(t, expected.GetName(), fileMeta.GetName())
			assert.NotEmpty(t, fileMeta.GetHash())
			assert.Equal(t, expected.GetPermissions(), fileMeta.GetPermissions())
			assert.NotNil(t, fileMeta.GetModifiedTime())

			if file != nil {
				helpers.RemoveFileWithErrorCheck(t, file.Name())
			}
		})
	}
}

func TestGetPermissions(t *testing.T) {
	file := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "get_permissions_test.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	info, err := os.Stat(file.Name())
	require.NoError(t, err)

	permissions := Permissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}

func Test_GenerateConfigVersion(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    []*mpi.File
	}{
		{
			name:     "Test 1: empty file slice",
			input:    []*mpi.File{},
			expected: GenerateHash([]byte{}),
		},
		{
			name: "Test 2: one file",
			input: []*mpi.File{
				{
					FileMeta: &mpi.FileMeta{
						Name: "file1",
						Hash: "hash1",
					},
					Action: nil,
				},
			},
			expected: GenerateHash([]byte("hash1")),
		},
		{
			name: "Test 3: multiple files",
			input: []*mpi.File{
				{
					FileMeta: &mpi.FileMeta{
						Name: "file1",
						Hash: "hash1",
					},
					Action: nil,
				},
				{
					FileMeta: &mpi.FileMeta{
						Name: "file2",
						Hash: "hash2",
					},
					Action: nil,
				},
			},
			expected: func() string {
				hashes := "hash1hash2"
				return GenerateHash([]byte(hashes))
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateConfigVersion(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateConfigVersion(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    []byte
	}{
		{
			name:     "Test 1: empty byte slice",
			input:    []byte{},
			expected: uuid.NewMD5(uuid.Nil, []byte("")).String(),
		},
		{
			name:     "Test 2: non-empty byte slice",
			input:    []byte("test"),
			expected: uuid.NewMD5(uuid.Nil, []byte("test")).String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHash(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateHash(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvertIpBytes(t *testing.T) {
	tests := []struct {
		input    []net.IP
		expected []string
	}{
		{
			input: []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(10, 0, 0, 1)},
			expected: []string{
				"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff\xff\xc0\xa8\x00\x01",
				"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff\xff\n\x00\x00\x01",
			},
		},
		{
			input:    []net.IP{net.ParseIP("2001:0db8::68")},
			expected: []string{" \x01\r\xb8\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00h"},
		},
		{
			input:    []net.IP{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := convertIPBytes(test.input)
		for i := range result {
			assert.Equal(t, test.expected[i], result[i])
		}
	}
}

func TestConvertX509SignatureAlgorithm(t *testing.T) {
	tests := []struct {
		input    x509.SignatureAlgorithm
		expected mpi.SignatureAlgorithm
	}{
		{x509.MD2WithRSA, mpi.SignatureAlgorithm_MD2_WITH_RSA},
		{x509.MD5WithRSA, mpi.SignatureAlgorithm_MD5_WITH_RSA},
		{x509.SHA1WithRSA, mpi.SignatureAlgorithm_SHA1_WITH_RSA},
		{x509.SHA256WithRSA, mpi.SignatureAlgorithm_SHA256_WITH_RSA},
		{x509.SHA384WithRSA, mpi.SignatureAlgorithm_SHA384_WITH_RSA},
		{x509.SHA512WithRSA, mpi.SignatureAlgorithm_SHA512_WITH_RSA},
		{x509.DSAWithSHA1, mpi.SignatureAlgorithm_DSA_WITH_SHA1},
		{x509.DSAWithSHA256, mpi.SignatureAlgorithm_DSA_WITH_SHA256},
		{x509.ECDSAWithSHA1, mpi.SignatureAlgorithm_ECDSA_WITH_SHA1},
		{x509.ECDSAWithSHA256, mpi.SignatureAlgorithm_ECDSA_WITH_SHA256},
		{x509.ECDSAWithSHA384, mpi.SignatureAlgorithm_ECDSA_WITH_SHA384},
		{x509.ECDSAWithSHA512, mpi.SignatureAlgorithm_ECDSA_WITH_SHA512},
		{x509.SHA256WithRSAPSS, mpi.SignatureAlgorithm_SHA256_WITH_RSA_PSS},
		{x509.SHA384WithRSAPSS, mpi.SignatureAlgorithm_SHA384_WITH_RSA_PSS},
		{x509.SHA512WithRSAPSS, mpi.SignatureAlgorithm_SHA512_WITH_RSA_PSS},
		{x509.PureEd25519, mpi.SignatureAlgorithm_PURE_ED25519},
		{x509.UnknownSignatureAlgorithm, mpi.SignatureAlgorithm_SIGNATURE_ALGORITHM_UNKNOWN},
	}

	for _, test := range tests {
		t.Run(test.input.String(), func(t *testing.T) {
			assert.Equal(t, test.expected, convertX509SignatureAlgorithm(test.input))
		})
	}
}

func TestMarshalAndUnmarshal(t *testing.T) {
	json := `{"messageMeta":{"messageId":"f0bc6a38-19d0-431d-a83e-75b2492b4d8f",
"correlationId":"e02df015-d0f3-4b4f-8505-a99ca3a675cf","timestamp":"2024-11-26T15:37:20.329251Z"},
"overview":{"files":[{"fileMeta":{"name":"/tmp/nginx-certs/nginx-repo.crt",
"hash":"077a855b-249e-321e-8430-10ffbc7159c5","modifiedTime":"2023-08-10T13:33:05.179811079Z",
"permissions":"0644","size":"1534","certificateMeta":{"serialNumber":"3894390349439439==",
"issuer":{"country":["US"],"organization":["F5 Networks, Inc."],"organizationalUnit":["Certificate Authority"],
"locality":["Seattle"],"province":["Washington"],"commonName":"Issuing Certificate Authority"},
"subject":{"commonName":"F5-A-S00009507"},"sans":{},"dates":{"notBefore":"1640105875","notAfter":"1734652800"},
"signatureAlgorithm":"SHA256_WITH_RSA","publicKeyAlgorithm":"RSA"}}},
{"fileMeta":{"name":"/opt/homebrew/etc/nginx/mime.types",
"hash":"91fca1c4-63b3-3525-ba0f-f001250576ed","modifiedTime":"2024-05-29T14:30:32Z",
"permissions":"0644","size":"5349"}},{"fileMeta":{"name":"/opt/homebrew/etc/nginx/nginx.conf",
"hash":"6f23df4d-82bd-3f72-89b4-e2681be219c1","modifiedTime":"2024-11-22T16:27:53.155484484Z",
"permissions":"0644","size":"1379"}}],"configVersion":
{"instanceId":"fc3773d1-857a-3a7d-86de-ef7e61339a82",
"version":"cfa04353-ff05-30a7-a13d-3b97de7319aa"}}}`

	t.Run("Test 1: Update Overview Request", func(t *testing.T) {
		var protoFromJSON mpi.UpdateOverviewRequest
		pb := protojson.UnmarshalOptions{DiscardUnknown: true, AllowPartial: true}
		unmarshalErr := pb.Unmarshal([]byte(json), &protoFromJSON)
		if unmarshalErr != nil {
			t.Fatalf("Failed to unmarshal embedded JSON: %v", unmarshalErr)
		}

		// Re-marshal to ensure consistency
		marshaledJSON, errMarshaledJSON := protojson.Marshal(&protoFromJSON)
		if errMarshaledJSON != nil {
			t.Fatalf("Failed to marshal struct back to JSON: %v", errMarshaledJSON)
		}

		// Re-parse marshaled JSON to validate round-trip correctness
		var parsedBack mpi.UpdateOverviewRequest
		if parsedBackErr := protojson.Unmarshal(marshaledJSON, &parsedBack); parsedBackErr != nil {
			t.Fatalf("Failed to parse back marshaled JSON: %v", parsedBackErr)
		}

		// Compare structs to ensure equality
		if diff := cmp.Diff(&protoFromJSON, &parsedBack, protocmp.Transform()); diff != "" {
			t.Errorf("Round-trip parsing mismatch (-want +got):\n%s", diff)
		}
	})
}
