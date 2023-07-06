/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/nginx/agent/sdk/v2/zip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultConfFileContentsString = `daemon            off;
		worker_processes  2;
		user              www-data;

		events {
			use           epoll;
			worker_connections  128;
		}

		error_log         /tmp/testdata/logs/error.log info;
				
		http {
			ssl_certificate     /tmp/testdata/nginx/ca.crt;

			server {
				server_name   localhost;
				listen        127.0.0.1:80;

				error_page    500 502 503 504  /50x.html;

				location      / {
					root      %s;
				}

				location      /duplicate-root-directory {
					root      %s;
				}

				location      /not-allowed {
					root      /not/allowed/root/directory/;
				}	
			}

			access_log    /tmp/testdata/logs/access2.log  combined;
		}
	`
	confFileContentsString = `daemon            off;
		worker_processes  2;
		user              www-data;
		
		events {
			use           epoll;
			worker_connections  128;
		}
		
		include %s;
	`
)

func TestNewConfigApply(t *testing.T) {
	tmpDir := t.TempDir()
	rootDirectory := path.Join(tmpDir, "root/")
	require.NoError(t, os.Mkdir(rootDirectory, os.ModePerm))

	rootFile1 := path.Join(rootDirectory, "root1.html")
	require.NoError(t, os.WriteFile(rootFile1, []byte{}, 0o644))

	rootFile2 := path.Join(rootDirectory, "root2.html")
	require.NoError(t, os.WriteFile(rootFile2, []byte{}, 0o644))

	rootFile3 := path.Join(rootDirectory, "root3.html")
	require.NoError(t, os.WriteFile(rootFile3, []byte{}, 0o644))

	emptyConfFile := path.Join(tmpDir, "empty_nginx.conf")
	require.NoError(t, os.WriteFile(emptyConfFile, []byte{}, 0o644))

	defaultConfFile := path.Join(tmpDir, "default_nginx.conf")
	defaultConfFileContent := fmt.Sprintf(defaultConfFileContentsString, rootDirectory, rootDirectory)
	require.NoError(t, os.WriteFile(defaultConfFile, []byte(defaultConfFileContent), 0o644))

	confFile := path.Join(tmpDir, "nginx.conf")
	confFileContent := fmt.Sprintf(confFileContentsString, defaultConfFile)
	require.NoError(t, os.WriteFile(confFile, []byte(confFileContent), 0o644))

	testCases := []struct {
		name                string
		confFile            string
		allowedDirectories  map[string]struct{}
		ignoreDirectives    []string
		expectedConfigApply *ConfigApply
		expectError         bool
	}{
		{
			name:     "config file present",
			confFile: confFile,
			allowedDirectories: map[string]struct{}{
				tmpDir: {},
			},
			ignoreDirectives: []string{},
			expectedConfigApply: &ConfigApply{
				existing: map[string]struct{}{
					defaultConfFile: {},
					confFile:        {},
					rootFile1:       {},
					rootFile2:       {},
					rootFile3:       {},
				},
				notExists:    map[string]struct{}{},
				notExistDirs: map[string]struct{}{},
			},
			expectError: false,
		},
		{
			name:               "no config file present",
			confFile:           "",
			allowedDirectories: map[string]struct{}{},
			ignoreDirectives:   []string{},
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{},
				notExistDirs: map[string]struct{}{},
			},
			expectError: false,
		},
		{
			name:               "empty config file present",
			confFile:           emptyConfFile,
			allowedDirectories: map[string]struct{}{},
			ignoreDirectives:   []string{},
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{},
				notExistDirs: map[string]struct{}{},
			},
			expectError: false,
		},
		{
			name:               "unknown config file present",
			confFile:           "/tmp/unknown.conf",
			allowedDirectories: map[string]struct{}{},
			ignoreDirectives:   []string{},
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{},
				notExistDirs: map[string]struct{}{},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configApply, err := NewConfigApplyWithIgnoreDirectives(tc.confFile, tc.allowedDirectories, tc.ignoreDirectives)
			assert.Equal(t, tc.expectedConfigApply.existing, configApply.GetExisting())
			assert.Equal(t, tc.expectedConfigApply.notExists, configApply.GetNotExists())
			assert.Equal(t, tc.expectedConfigApply.notExistDirs, configApply.GetNotExistDirs())
			if tc.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestConfigApplyMarkAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	unknownFile := path.Join(tmpDir, "unknown.conf")
	knownFile := path.Join(tmpDir, "known.conf")
	unknownFileUnknownDir := path.Join(tmpDir, "/unknown/unknown.conf")
	unknownFileUnknownNestedDirs := path.Join(tmpDir, "/unknown/nested/unknown.conf")

	require.NoError(t, os.WriteFile(knownFile, []byte{}, 0o644))

	writer, err := zip.NewWriter("/")
	require.NoError(t, err)

	testCases := []struct {
		name                string
		file                string
		expectedConfigApply *ConfigApply
	}{
		{
			name: "file doesn't exist",
			file: unknownFile,
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{unknownFile: {}},
				notExistDirs: map[string]struct{}{},
				writer:       writer,
			},
		},
		{
			name: "file exists",
			file: knownFile,
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{},
				notExistDirs: map[string]struct{}{},
			},
		},
		{
			name: "file doesn't exist and dir doesn't exist",
			file: unknownFileUnknownDir,
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{unknownFileUnknownDir: {}},
				notExistDirs: map[string]struct{}{path.Dir(unknownFileUnknownDir): {}},
			},
		},
		{
			name: "file doesn't exist and nested new dirs don't exist",
			file: unknownFileUnknownNestedDirs,
			expectedConfigApply: &ConfigApply{
				existing:     map[string]struct{}{},
				notExists:    map[string]struct{}{unknownFileUnknownNestedDirs: {}},
				notExistDirs: map[string]struct{}{path.Dir(unknownFileUnknownDir): {}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configApply := &ConfigApply{
				existing:     make(map[string]struct{}),
				notExists:    make(map[string]struct{}),
				notExistDirs: make(map[string]struct{}),
				writer:       writer,
			}

			assert.NoError(t, configApply.MarkAndSave(tc.file))
			assert.Equal(t, tc.expectedConfigApply.existing, configApply.GetExisting())
			assert.Equal(t, tc.expectedConfigApply.notExists, configApply.GetNotExists())
			assert.Equal(t, tc.expectedConfigApply.notExistDirs, configApply.GetNotExistDirs())
		})
	}
}

func TestConfigApplyCompleteAndRollback(t *testing.T) {
	tmpDir := t.TempDir()
	rootDirectory := path.Join(tmpDir, "root/")
	require.NoError(t, os.Mkdir(rootDirectory, os.ModePerm))

	rootFile1 := path.Join(rootDirectory, "root1.html")
	require.NoError(t, os.WriteFile(rootFile1, []byte{}, 0o644))

	rootFile2 := path.Join(rootDirectory, "root2.html")
	require.NoError(t, os.WriteFile(rootFile2, []byte{}, 0o644))

	rootFile3 := path.Join(rootDirectory, "root3.html")
	require.NoError(t, os.WriteFile(rootFile3, []byte{}, 0o644))

	defaultConfFile := path.Join(tmpDir, "default_nginx.conf")
	defaultConfFileContent := fmt.Sprintf(defaultConfFileContentsString, rootDirectory, rootDirectory)
	require.NoError(t, os.WriteFile(defaultConfFile, []byte(defaultConfFileContent), 0o644))

	confFile := path.Join(tmpDir, "nginx.conf")
	confFileContent := fmt.Sprintf(confFileContentsString, defaultConfFile)
	require.NoError(t, os.WriteFile(confFile, []byte(confFileContent), 0o644))

	allowedDirectories := map[string]struct{}{tmpDir: {}}
	ignoreDirectives := []string{}

	configApply, err := NewConfigApplyWithIgnoreDirectives(confFile, allowedDirectories, ignoreDirectives)
	assert.Equal(t, 5, len(configApply.GetExisting()))
	assert.Nil(t, err)

	// Only mark and save the config files
	assert.NoError(t, configApply.MarkAndSave(defaultConfFile))
	assert.NoError(t, configApply.MarkAndSave(confFile))

	// MarkAndSave an unknown file that does not exist, then create the file afterwards
	unknownConfFile := path.Join(tmpDir, "unknown.conf")
	assert.NoError(t, configApply.MarkAndSave(unknownConfFile))
	assert.NoError(t, os.WriteFile(unknownConfFile, []byte{}, 0o644))

	// Verify that only root files are deleted
	assert.NoError(t, configApply.Complete())
	assert.FileExists(t, defaultConfFile)
	assert.FileExists(t, confFile)
	assert.FileExists(t, unknownConfFile)
	assert.NoFileExists(t, rootFile1)
	assert.NoFileExists(t, rootFile2)
	assert.NoFileExists(t, rootFile3)

	// Delete the config files
	assert.NoError(t, os.Remove(defaultConfFile))
	assert.NoError(t, os.Remove(confFile))

	// Verify that the rollback recreates the deleted config files and removes the unknown file
	assert.NoError(t, configApply.Rollback(errors.New("error")))
	assert.FileExists(t, defaultConfFile)
	assert.FileExists(t, confFile)
	assert.NoFileExists(t, unknownConfFile)
	assert.NoFileExists(t, rootFile1)
	assert.NoFileExists(t, rootFile2)
	assert.NoFileExists(t, rootFile3)
}
