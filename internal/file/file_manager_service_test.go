// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/pkg/files"
	"google.golang.org/protobuf/types/known/timestamppb"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManagerService_UpdateOverview(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/etc/nginx/nginx.conf", "")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
	fileManagerService.SetIsConnected(true)

	err := fileManagerService.UpdateOverview(ctx, "123", []*mpi.File{
		{
			FileMeta: fileMeta,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, fakeFileServiceClient.UpdateOverviewCallCount())
}

func TestFileManagerService_UpdateFile(t *testing.T) {
	tests := []struct {
		name   string
		isCert bool
	}{
		{
			name:   "non-cert",
			isCert: false,
		},
		{
			name:   "cert",
			isCert: true,
		},
	}

	tempDir := os.TempDir()

	for _, test := range tests {
		ctx := context.Background()

		testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")

		var fileMeta *mpi.FileMeta
		if test.isCert {
			fileMeta = protos.CertMeta(testFile.Name(), "")
		} else {
			fileMeta = protos.FileMeta(testFile.Name(), "")
		}

		fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
		fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
		fileManagerService.SetIsConnected(true)

		err := fileManagerService.UpdateFile(ctx, "123", &mpi.File{FileMeta: fileMeta})

		require.NoError(t, err)
		assert.Equal(t, 1, fakeFileServiceClient.UpdateFileCallCount())

		helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	}
}

func TestFileManagerService_ConfigApply_Add(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "nginx.conf")
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)

	overview := protos.FileOverview(filePath, fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.fileActions[filePath], overview.GetFiles()[0])
}

func TestFileManagerService_ConfigApply_Update(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	previousFileContent := []byte("some test data")
	previousFileHash := files.GenerateHash(previousFileContent)
	fileHash := files.GenerateHash(fileContent)
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(previousFileContent)
	require.NoError(t, writeErr)
	defer helpers.RemoveFileWithErrorCheck(t, tempFile.Name())

	filesOnDisk := map[string]*mpi.File{
		tempFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         tempFile.Name(),
				Hash:         previousFileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0640",
				Size:         0,
			},
		},
	}

	overview := protos.FileOverview(tempFile.Name(), fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)
	fileManagerService.UpdateCurrentFilesOnDisk(filesOnDisk)

	request := protos.CreateConfigApplyRequest(overview)

	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	data, readErr := os.ReadFile(tempFile.Name())
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.rollbackFileContents[tempFile.Name()], previousFileContent)
	assert.Equal(t, fileManagerService.fileActions[tempFile.Name()], overview.GetFiles()[0])
}

func TestFileManagerService_ConfigApply_Delete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	fileContent := []byte("location /test {\n return 200 \"Test location\\n\";\n}")
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(fileContent)
	require.NoError(t, writeErr)

	tempFile2 := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	overview := protos.FileOverview(tempFile2.Name(), files.GenerateHash(fileContent))

	filesOnDisk := map[string]*mpi.File{
		tempFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         tempFile.Name(),
				Hash:         files.GenerateHash(fileContent),
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0640",
				Size:         0,
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)
	fileManagerService.UpdateCurrentFilesOnDisk(filesOnDisk)

	request := protos.CreateConfigApplyRequest(overview)

	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)

	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.NoFileExists(t, tempFile.Name())
	assert.Equal(t, fileManagerService.rollbackFileContents[tempFile.Name()], fileContent)
	assert.Equal(t, fileManagerService.fileActions[tempFile.Name()], filesOnDisk[tempFile.Name()])
	assert.Equal(t, model.OK, writeStatus)
}

func TestFileManagerService_checkAllowedDirectory(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	allowedFiles := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:         "/tmp/local/etc/nginx/allowedDirPath",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
			Action: nil,
		},
	}

	notAllowed := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:         "/not/allowed/dir/path",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
			Action: nil,
		},
	}

	err := fileManagerService.checkAllowedDirectory(allowedFiles)
	require.NoError(t, err)
	err = fileManagerService.checkAllowedDirectory(notAllowed)
	require.Error(t, err)
}

func TestFileManagerService_ClearCache(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	filesCache := map[string]*mpi.File{
		"file/path/test.conf": {
			FileMeta: &mpi.FileMeta{
				Name:         "file/path/test.conf",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
		},
	}

	contentsCache := map[string][]byte{
		"file/path/test.conf": []byte("some test data"),
	}

	fileManagerService.fileActions = filesCache
	fileManagerService.rollbackFileContents = contentsCache
	assert.NotEmpty(t, fileManagerService.fileActions)
	assert.NotEmpty(t, fileManagerService.rollbackFileContents)

	fileManagerService.ClearCache()

	assert.Empty(t, fileManagerService.fileActions)
	assert.Empty(t, fileManagerService.rollbackFileContents)
}

func TestFileManagerService_Rollback(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	addAction := mpi.File_FILE_ACTION_ADD
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	unspecifiedAction := mpi.File_FILE_ACTION_UNSPECIFIED

	deleteFilePath := filepath.Join(tempDir, "nginx_delete.conf")

	newFileContent := []byte("location /test {\n    return 200 \"This config needs to be rolled back\\n\";\n}")
	oldFileContent := []byte("location /test {\n    return 200 \"This is the saved config\\n\";\n}")
	fileHash := files.GenerateHash(newFileContent)

	addFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_add.conf")
	_, writeErr := addFile.Write(newFileContent)
	require.NoError(t, writeErr)

	updateFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	_, writeErr = updateFile.Write(newFileContent)
	require.NoError(t, writeErr)

	filesCache := map[string]*mpi.File{
		addFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         addFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &addAction,
		},
		updateFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         updateFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &updateAction,
		},
		deleteFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         deleteFilePath,
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &deleteAction,
		},
		"unspecified/file/test.conf": {
			FileMeta: &mpi.FileMeta{
				Name:         "unspecified/file/test.conf",
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &unspecifiedAction,
		},
	}

	fileContentCache := map[string][]byte{
		deleteFilePath:    oldFileContent,
		updateFile.Name(): oldFileContent,
	}

	instanceID := protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
	fileManagerService.rollbackFileContents = fileContentCache
	fileManagerService.fileActions = filesCache

	err := fileManagerService.Rollback(ctx, instanceID)
	require.NoError(t, err)

	assert.NoFileExists(t, addFile.Name())
	assert.FileExists(t, deleteFilePath)
	updateData, readUpdateErr := os.ReadFile(updateFile.Name())
	require.NoError(t, readUpdateErr)
	assert.Equal(t, oldFileContent, updateData)

	deleteData, readDeleteErr := os.ReadFile(deleteFilePath)
	require.NoError(t, readDeleteErr)
	assert.Equal(t, oldFileContent, deleteData)

	defer helpers.RemoveFileWithErrorCheck(t, updateFile.Name())
	defer helpers.RemoveFileWithErrorCheck(t, deleteFilePath)
}

func TestFileManagerService_DetermineFileActions(t *testing.T) {
	// Go doesn't allow address of numeric constant
	addAction := mpi.File_FILE_ACTION_ADD
	updateAction := mpi.File_FILE_ACTION_UPDATE
	deleteAction := mpi.File_FILE_ACTION_DELETE

	tempDir := os.TempDir()

	deleteTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_delete.conf")
	defer helpers.RemoveFileWithErrorCheck(t, deleteTestFile.Name())
	fileContent, readErr := os.ReadFile("../../test/config/nginx/nginx.conf")
	require.NoError(t, readErr)
	err := os.WriteFile(deleteTestFile.Name(), fileContent, 0o600)
	require.NoError(t, err)

	updateTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	defer helpers.RemoveFileWithErrorCheck(t, updateTestFile.Name())
	updatedFileContent := []byte("test update file")
	updateErr := os.WriteFile(updateTestFile.Name(), updatedFileContent, 0o600)
	require.NoError(t, updateErr)

	addTestFileName := tempDir + "/nginx_add.conf"

	tests := []struct {
		expectedError   error
		modifiedFiles   map[string]*mpi.File
		currentFiles    map[string]*mpi.File
		expectedCache   map[string]*mpi.File
		expectedContent map[string][]byte
		name            string
	}{
		{
			name: "Test 1: Add, Update & Delete Files",
			modifiedFiles: map[string]*mpi.File{
				addTestFileName: {
					FileMeta: protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
				},
			},
			currentFiles: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
				},
			},
			expectedCache: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
					Action:   &deleteAction,
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
					Action:   &updateAction,
				},
				addTestFileName: {
					FileMeta: protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
					Action:   &addAction,
				},
			},
			expectedContent: map[string][]byte{
				deleteTestFile.Name(): fileContent,
				updateTestFile.Name(): updatedFileContent,
			},
			expectedError: nil,
		},
		{
			name: "Test 2: Files same as on disk",
			modifiedFiles: map[string]*mpi.File{
				addTestFileName: {
					FileMeta: protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
				},
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
				},
			},
			currentFiles: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
				},
				addTestFileName: {
					FileMeta: protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
				},
			},
			expectedCache:   make(map[string]*mpi.File),
			expectedContent: make(map[string][]byte),
			expectedError:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
			diff, contents, fileActionErr := fileManagerService.DetermineFileActions(test.currentFiles,
				test.modifiedFiles)

			require.NoError(tt, fileActionErr)
			assert.Equal(tt, test.expectedContent, contents)
			assert.Equal(tt, test.expectedCache, diff)
		})
	}
}

func TestFileManagerService_fileActions(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	addAction := mpi.File_FILE_ACTION_ADD
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	unspecifiedAction := mpi.File_FILE_ACTION_UNSPECIFIED

	addFilePath := filepath.Join(tempDir, "nginx_add.conf")
	unspecifiedFilePath := "unspecified/file/test.conf"

	newFileContent := []byte("location /test {\n    return 200 \"This config needs to be rolled back\\n\";\n}")
	oldFileContent := []byte("location /test {\n    return 200 \"This is the saved config\\n\";\n}")
	fileHash := files.GenerateHash(newFileContent)

	deleteFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_delete.conf")
	_, writeErr := deleteFile.Write(oldFileContent)
	require.NoError(t, writeErr)

	updateFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	_, writeErr = updateFile.Write(oldFileContent)
	require.NoError(t, writeErr)

	filesCache := map[string]*mpi.File{
		addFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         addFilePath,
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &addAction,
		},
		updateFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         updateFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &updateAction,
		},
		deleteFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         deleteFile.Name(),
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &deleteAction,
		},
		unspecifiedFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         unspecifiedFilePath,
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &unspecifiedAction,
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: newFileContent,
		},
	}, nil)
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	fileManagerService.fileActions = filesCache

	actionErr := fileManagerService.executeFileActions(ctx)
	require.NoError(t, actionErr)

	assert.FileExists(t, addFilePath)
	assert.NoFileExists(t, deleteFile.Name())
	assert.NoFileExists(t, unspecifiedFilePath)
	updateData, readUpdateErr := os.ReadFile(updateFile.Name())
	require.NoError(t, readUpdateErr)
	assert.Equal(t, newFileContent, updateData)

	defer helpers.RemoveFileWithErrorCheck(t, updateFile.Name())
	defer helpers.RemoveFileWithErrorCheck(t, addFilePath)
}

func TestParseX509Certificates(t *testing.T) {
	tests := []struct {
		certName       string
		certContent    string
		name           string
		expectedSerial string
	}{
		{
			name:           "Test 1: generated cert",
			certName:       "public_cert",
			certContent:    "",
			expectedSerial: "123123",
		},
		{
			name:     "Test 2: open ssl cert",
			certName: "open_ssl_cert",
			certContent: `-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUR+YGgRHhYwotFyBOvSc1KD9d45kwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNDExMjcxNTM0MDZaFw0yNDEy
MjcxNTM0MDZaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDnDDVGflbZ3dmuQJj+8QuJIQ8lWjVGYhlsFI4AGFTX
9VfYOqJEPyuMRuSj2eN7C/mR4yTJSggnv0kFtjmeGh2keNdmb4R/0CjYWZVl/Na6
cAfldB8v2+sm0LZ/OD9F9CbnYB95takPOZq3AP5kUA+qlFYzroqXsxJKvZF6dUuI
+kTOn5pWD+eFmueFedOz1aucOvblUJLueVZnvAbIrBoyaulw3f2kjk0J1266nFMb
s72AvjyYbOXbyur3BhPThCaOeqMGggDmFslZ4pBgQFWUeFvmqJMFzf1atKTWlbj7
Mj+bNKNs4xvUuNhqd/F99Pz2Fe0afKbTHK83hqgSHKbtAgMBAAGjUzBRMB0GA1Ud
DgQWBBQq0Bzde0bl9CFb81LrvFfdWlY7hzAfBgNVHSMEGDAWgBQq0Bzde0bl9CFb
81LrvFfdWlY7hzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAo
8GXvwRa0M0D4x4Lrj2K57FxH4ECNBnAqWlh3Ce9LEioL2CYaQQw6I2/FsnTk8TYY
WgGgXMEyA6OeOXvwxWjSllK9+D2ueTMhNRO0tYMUi0kDJqd9EpmnEcSWIL2G2SNo
BWQjqEoEKFjvrgx6h13AtsFlpdURoVtodrtnUrXp1r4wJvljC2qexoNfslhpbqsT
X/vYrzgKRoKSUWUt1ejKTntrVuaJK4NMxANOTTjIXgxyoV3YcgEmL9KzribCqILi
p79Nno9d+kovtX5VKsJ5FCcPw9mEATgZDOQ4nLTk/HHG6bwtpubp6Zb7H1AjzBkz
rQHX6DP4w6IwZY8JB8LS
-----END CERTIFICATE-----`,
			expectedSerial: "410468082718062724391949173062901619571168240537",
		},
	}

	tempDir := os.TempDir()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var certBytes []byte
			var certPath string

			if test.certContent == "" {
				_, certBytes = helpers.GenerateSelfSignedCert(t)
				certContents := helpers.Cert{
					Name:     fmt.Sprintf("%s.pem", test.certName),
					Type:     "CERTIFICATE",
					Contents: certBytes,
				}
				certPath = helpers.WriteCertFiles(t, tempDir, certContents)
			} else {
				certPath = fmt.Sprintf("%s%c%s", tempDir, os.PathSeparator, test.certName)
				err := os.WriteFile(certPath, []byte(test.certContent), 0o600)
				require.NoError(t, err)
			}

			certFileMeta, certFileMetaErr := files.FileMetaWithCertificate(certPath)
			require.NoError(t, certFileMetaErr)

			assert.Equal(t, test.expectedSerial, certFileMeta.GetCertificateMeta().GetSerialNumber())
		})
	}
}
