// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
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
				},
				{
					FileMeta: &mpi.FileMeta{
						Name: "file2",
						Hash: "hash2",
					},
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
	hash1 := sha256.New()
	hash2 := sha256.New()
	hash1.Write([]byte(""))
	hash2.Write([]byte("test"))
	tests := []struct {
		name     string
		expected string
		input    []byte
	}{
		{
			name:     "Test 1: empty byte slice",
			input:    []byte{},
			expected: base64.StdEncoding.EncodeToString(hash1.Sum(nil)),
		},
		{
			name:     "Test 2: non-empty byte slice",
			input:    []byte("test"),
			expected: base64.StdEncoding.EncodeToString(hash2.Sum(nil)),
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

func TestConvertIpToString(t *testing.T) {
	tests := []struct {
		input    []net.IP
		expected []string
	}{
		{
			input: []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(10, 0, 0, 1)},
			expected: []string{
				"192.168.0.1",
				"10.0.0.1",
			},
		},
		{
			input:    []net.IP{net.ParseIP("2001:0db8::68")},
			expected: []string{"2001:db8::68"},
		},
		{
			input:    []net.IP{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		result := convertIPToString(test.input)
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
