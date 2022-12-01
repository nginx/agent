/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cgroup

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadLines(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name      string
		file      string
		lines     []string
		errorType error
	}{
		{"file exists", localDirectory + "/../testdata/good_data/v1/cpuacct/cpuacct.stat", []string{"user 5760", "system 1753"}, nil},
		{"no file", localDirectory + "/unknown/file.stat", []string{}, &os.PathError{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			lines, err := ReadLines(test.file)

			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, len(test.lines), len(lines))
			for index, line := range lines {
				assert.Equal(tt, test.lines[index], line)
			}
		})
	}
}

func TestReadSingleValueCgroupFile(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name      string
		file      string
		value     string
		errorType error
	}{
		{"file exists", localDirectory + "/../testdata/good_data/v1/memory/memory.usage_in_bytes", "392151040", nil},
		{"value in file is max", localDirectory + "/../testdata/good_data_no_limits/v2/memory.max", "max", nil},
		{"no file", localDirectory + "/unknown/memory.usage_in_bytes", "", &os.PathError{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			value, err := ReadSingleValueCgroupFile(test.file)
			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, test.value, value)
		})
	}
}

func TestReadIntegerValueCgroupFile(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name      string
		file      string
		value     uint64
		errorType error
	}{
		{"file exists", localDirectory + "/../testdata/good_data/v1/memory/memory.usage_in_bytes", 392151040, nil},
		{"no file", localDirectory + "/unknown/memory.usage_in_bytes", 0, &os.PathError{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			value, err := ReadIntegerValueCgroupFile(test.file)
			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, test.value, value)
		})
	}
}

func TestIsCgroupV2(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name     string
		basePath string
		value    bool
	}{
		{"v1", localDirectory + "/../testdata/good_data/v1/", false},
		{"v2", localDirectory + "/../testdata/good_data/v2/", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			value := IsCgroupV2(test.basePath)

			// Assert result
			assert.Equal(tt, test.value, value)
		})
	}
}

func TestGetV1DefaultMaxValue(t *testing.T) {
	pageSize = 65536
	assert.Equal(t, "9223372036854710272", GetV1DefaultMaxValue())
}
