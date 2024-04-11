// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfo_IsContainer(t *testing.T) {
	containerSpecificFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), ".dockerenv")
	defer helpers.RemoveFileWithErrorCheck(t, containerSpecificFile.Name())

	selfCgroupFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "cgroup")
	defer helpers.RemoveFileWithErrorCheck(t, selfCgroupFile.Name())
	err := os.WriteFile(selfCgroupFile.Name(), []byte(docker), os.ModeAppend)
	require.NoError(t, err)

	emptySelfCgroupFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "cgroup")
	defer helpers.RemoveFileWithErrorCheck(t, emptySelfCgroupFile.Name())

	tests := []struct {
		name                   string
		containerSpecificFiles []string
		selfCgroupLocation     string
		expected               bool
	}{
		{
			name:                   "Test 1: no container references",
			containerSpecificFiles: []string{},
			selfCgroupLocation:     "/unknown/path",
			expected:               false,
		},
		{
			name:                   "Test 2: container specific file found",
			containerSpecificFiles: []string{containerSpecificFile.Name()},
			selfCgroupLocation:     "/unknown/path",
			expected:               true,
		},
		{
			name:                   "Test 3: container reference in self cgroup file",
			containerSpecificFiles: []string{},
			selfCgroupLocation:     selfCgroupFile.Name(),
			expected:               true,
		},
		{
			name:                   "Test 4: no container reference in self cgroup file",
			containerSpecificFiles: []string{},
			selfCgroupLocation:     emptySelfCgroupFile.Name(),
			expected:               false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			info := NewInfo()
			info.containerSpecificFiles = test.containerSpecificFiles
			info.selfCgroupLocation = test.selfCgroupLocation

			assert.Equal(tt, test.expected, info.IsContainer())
		})
	}
}
