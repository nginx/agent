// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gocognit,revive,govet // cognitive complexity is 22
func TestFileManagerService_downloadExternalFiles(t *testing.T) {
	type tc struct {
		allowedDomains      []string
		expectContent       []byte
		name                string
		expectHeaderETag    string
		expectHeaderLastMod string
		expectErrContains   string
		handler             http.HandlerFunc
		maxBytes            int
		expectError         bool
		expectTempFile      bool
	}

	tests := []tc{
		{
			name: "Test 1: Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", "test-etag")
				w.Header().Set("Last-Modified", time.RFC1123)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file content"))
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      true,
			expectContent:       []byte("external file content"),
			expectHeaderETag:    "test-etag",
			expectHeaderLastMod: time.RFC1123,
		},
		{
			name: "Test 2: NotModified",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotModified)
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      false,
			expectContent:       nil,
			expectHeaderETag:    "",
			expectHeaderLastMod: "",
		},
		{
			name: "Test 3: NotAllowedDomain",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file content"))
			},
			allowedDomains:    []string{"not-the-host"},
			maxBytes:          0,
			expectError:       true,
			expectErrContains: "not in the allowed domains",
			expectTempFile:    false,
		},
		{
			name: "Test 4: NotFound",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			allowedDomains:    nil,
			maxBytes:          0,
			expectError:       true,
			expectErrContains: "status code 404",
			expectTempFile:    false,
		},
		{
			name: "Test 5: ProxyWithConditionalHeaders",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// verify conditional headers from manifest are added
				if r.Header.Get("If-None-Match") != "manifest-test-etag" {
					http.Error(w, "missing If-None-Match", http.StatusBadRequest)
					return
				}
				if r.Header.Get("If-Modified-Since") != time.RFC1123 {
					http.Error(w, "missing If-Modified-Since", http.StatusBadRequest)
					return
				}
				w.Header().Set("ETag", "resp-etag")
				w.Header().Set("Last-Modified", time.RFC1123)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file via proxy"))
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      true,
			expectContent:       []byte("external file via proxy"),
			expectHeaderETag:    "resp-etag",
			expectHeaderLastMod: time.RFC1123,
			expectErrContains:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			tempDir := t.TempDir()
			fileName := filepath.Join(tempDir, "external.conf")

			ts := httptest.NewServer(test.handler)
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			require.NoError(t, err)
			host := u.Hostname()

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})

			eds := &config.ExternalDataSource{
				ProxyURL:       config.ProxyURL{URL: ""},
				AllowedDomains: []string{host},
				MaxBytes:       int64(test.maxBytes),
			}

			if test.allowedDomains != nil {
				eds.AllowedDomains = test.allowedDomains
			}

			if test.name == "Test 5: ProxyWithConditionalHeaders" {
				manifestFiles := map[string]*model.ManifestFile{
					fileName: {
						ManifestFileMeta: &model.ManifestFileMeta{
							Name:         fileName,
							ETag:         "manifest-test-etag",
							LastModified: time.RFC1123,
						},
					},
				}
				manifestJSON, mErr := json.MarshalIndent(manifestFiles, "", "  ")
				require.NoError(t, mErr)

				manifestFile, mErr := os.CreateTemp(tempDir, "manifest.json")
				require.NoError(t, mErr)
				_, mErr = manifestFile.Write(manifestJSON)
				require.NoError(t, mErr)
				_ = manifestFile.Close()

				fileManagerService.agentConfig.LibDir = tempDir
				fileManagerService.manifestFilePath = manifestFile.Name()

				eds.ProxyURL = config.ProxyURL{URL: ts.URL}
			}

			fileManagerService.agentConfig.ExternalDataSource = eds

			fileManagerService.fileActions = map[string]*model.FileCache{
				fileName: {
					File: &mpi.File{
						FileMeta:           &mpi.FileMeta{Name: fileName},
						ExternalDataSource: &mpi.ExternalDataSource{Location: ts.URL},
					},
					Action: model.ExternalFile,
				},
			}

			err = fileManagerService.downloadUpdatedFilesToTempLocation(ctx)

			if test.expectError {
				require.Error(t, err)
				if test.expectErrContains != "" {
					assert.Contains(t, err.Error(), test.expectErrContains)
				}
				_, statErr := os.Stat(tempFilePath(fileName))
				assert.True(t, os.IsNotExist(statErr))

				return
			}

			require.NoError(t, err)

			if test.expectTempFile {
				b, readErr := os.ReadFile(tempFilePath(fileName))
				require.NoError(t, readErr)
				assert.Equal(t, test.expectContent, b)

				h, ok := fileManagerService.externalFileHeaders[fileName]
				require.True(t, ok)
				assert.Equal(t, test.expectHeaderETag, h.ETag)
				assert.Equal(t, test.expectHeaderLastMod, h.LastModified)

				_ = os.Remove(tempFilePath(fileName))
			} else {
				_, statErr := os.Stat(tempFilePath(fileName))
				assert.True(t, os.IsNotExist(statErr))
			}
		})
	}
}

func TestFileManagerService_DownloadFileContent_MaxBytesLimit(t *testing.T) {
	ctx := context.Background()
	fms := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	// test server returns 10 bytes, we set MaxBytes to 4 and expect only 4 bytes returned
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "etag-1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	fms.agentConfig.ExternalDataSource = &config.ExternalDataSource{
		AllowedDomains: []string{u.Hostname()},
		MaxBytes:       4,
	}

	fileName := t.TempDir() + "/external.conf"
	file := &mpi.File{
		FileMeta:           &mpi.FileMeta{Name: fileName},
		ExternalDataSource: &mpi.ExternalDataSource{Location: ts.URL},
	}

	content, headers, err := fms.externalFileOperator.downloadFileContent(ctx, file)
	require.NoError(t, err)
	assert.Len(t, content, 4)
	assert.Equal(t, "etag-1", headers.ETag)
}

func TestFileManagerService_TestDownloadFileContent_InvalidProxyURL(t *testing.T) {
	ctx := context.Background()
	fms := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	downURL := "http://example.com/file"
	fms.agentConfig.ExternalDataSource = &config.ExternalDataSource{
		AllowedDomains: []string{"example.com"},
		ProxyURL:       config.ProxyURL{URL: "http://:"},
	}

	file := &mpi.File{
		FileMeta:           &mpi.FileMeta{Name: "/tmp/file"},
		ExternalDataSource: &mpi.ExternalDataSource{Location: downURL},
	}

	_, _, err := fms.externalFileOperator.downloadFileContent(ctx, file)
	require.Error(t, err)
	if !strings.Contains(err.Error(), "invalid proxy URL configured") &&
		!strings.Contains(err.Error(), "failed to execute download request") &&
		!strings.Contains(err.Error(), "proxyconnect") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileManagerService_IsDomainAllowed(t *testing.T) {
	type testCase struct {
		name           string
		url            string
		allowedDomains []string
		expected       bool
	}

	tests := []testCase{
		{
			name:           "Invalid URL (Percent)",
			url:            "http://%",
			allowedDomains: []string{"example.com"},
			expected:       false,
		},
		{
			name:           "Invalid URL (Empty Host)",
			url:            "http://",
			allowedDomains: []string{"example.com"},
			expected:       false,
		},
		{
			name:           "Empty Allowed List",
			url:            "http://example.com/path",
			allowedDomains: []string{""},
			expected:       false,
		},
		{
			name:           "Basic Match",
			url:            "http://example.com/path",
			allowedDomains: []string{"example.com"},
			expected:       true,
		},
		{
			name:           "Wildcard Subdomain Match",
			url:            "http://sub.example.com/path",
			allowedDomains: []string{"*.example.com"},
			expected:       true,
		},
	}

	fms := NewFileManagerService(&v1fakes.FakeFileServiceClient{}, types.AgentConfig(), &sync.RWMutex{})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := fms.externalFileOperator.isDomainAllowed(tc.url, tc.allowedDomains)
			assert.Equal(t, tc.expected, actual, "for URL: %s and domains: %v", tc.url, tc.allowedDomains)
		})
	}
}

func TestFileManagerService_IsMatchesWildcardDomain(t *testing.T) {
	type testCase struct {
		name     string
		hostname string
		pattern  string
		expected bool
	}

	tests := []testCase{
		{
			name:     "True Match - Subdomain",
			hostname: "sub.example.com",
			pattern:  "*.example.com",
			expected: true,
		},
		{
			name:     "True Match - Exact Base Domain",
			hostname: "example.com",
			pattern:  "*.example.com",
			expected: true,
		},
		{
			name:     "False Match - Bad Domain Suffix",
			hostname: "badexample.com",
			pattern:  "*.example.com",
			expected: false,
		},
		{
			name:     "False Match - No Wildcard Prefix",
			hostname: "test.com",
			pattern:  "google.com",
			expected: false,
		},
		{
			name:     "False Match - Different Suffix",
			hostname: "sub.anotherexample.com",
			pattern:  "*.example.com",
			expected: false,
		},
	}

	fms := NewFileManagerService(&v1fakes.FakeFileServiceClient{}, types.AgentConfig(), &sync.RWMutex{})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := fms.externalFileOperator.isMatchesWildcardDomain(tc.hostname, tc.pattern)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExternalFileOperator_validateDownloadedFile(t *testing.T) {
	// PNG signature bytes + some payload
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00}
	textBytes := []byte("hello world")

	tests := []struct {
		content     []byte
		name        string
		fileName    string
		errContains string
		allowedList []string
		expectErr   bool
	}{
		{
			name:      "Accept when extension matches detected MIME (text)",
			content:   textBytes,
			fileName:  "/tmp/file.txt",
			expectErr: false,
		},
		{
			name:        "Reject when extension maps to different MIME (html implies text/html but content is png)",
			content:     pngBytes,
			fileName:    "/tmp/file.html",
			expectErr:   true,
			errContains: "implies MIME",
		},
		{
			name:        "Allow when extension unknown but detected MIME is in allowed list",
			content:     pngBytes,
			fileName:    "/tmp/file.unknownext",
			allowedList: []string{"image/png"},
			expectErr:   false,
		},
		{
			name:        "Reject when detected MIME not in allowed list and extension has no known mapping",
			content:     pngBytes,
			fileName:    "/tmp/file.bin",
			allowedList: []string{".conf"},
			expectErr:   true,
			errContains: "implies MIME",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fms := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})
			fms.agentConfig.ExternalDataSource = &config.ExternalDataSource{
				AllowedFileTypes: tc.allowedList,
			}

			err := fms.externalFileOperator.validateDownloadedFile(tc.content, tc.fileName)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
