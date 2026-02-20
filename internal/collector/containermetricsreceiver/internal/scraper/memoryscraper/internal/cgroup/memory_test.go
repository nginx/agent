// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cgroup

import (
	"context"
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/require"
)

func TestVirtualMemoryStat(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		errorType         error
		name              string
		basePath          string
		virtualMemoryStat mem.VirtualMemoryStat
	}{
		{
			name:     "Test 1: v1 good data",
			basePath: localDirectory + "/../../../testdata/good_data/v1/",
			virtualMemoryStat: mem.VirtualMemoryStat{
				Total:       536870912,
				Free:        420200448,
				Available:   420200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 21,
			},
			errorType: nil,
		},
		{
			name:     "Test 2: v1 good data no limits",
			basePath: localDirectory + "/../../../testdata/good_data_no_limits/v1/",
			virtualMemoryStat: mem.VirtualMemoryStat{
				Total:       636870912,
				Free:        520200448,
				Available:   520200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 18,
			},
			errorType: nil,
		},
		{
			name:              "Test 3: v1 bad data",
			basePath:          localDirectory + "/../../../testdata/bad_data/v1/",
			virtualMemoryStat: mem.VirtualMemoryStat{},
			errorType:         &strconv.NumError{},
		},
		{
			name:     "Test 4: v2 good data",
			basePath: localDirectory + "/../../../testdata/good_data/v2/",
			virtualMemoryStat: mem.VirtualMemoryStat{
				Total:       536870912,
				Free:        420200448,
				Available:   420200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 21,
			},
			errorType: nil,
		},
		{
			name:     "Test 5: v2 good data no limits",
			basePath: localDirectory + "/../../../testdata/good_data_no_limits/v2/",
			virtualMemoryStat: mem.VirtualMemoryStat{
				Total:       636870912,
				Free:        520200448,
				Available:   520200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 18,
			},
			errorType: nil,
		},
		{
			name:              "Test 6: v2 bad data",
			basePath:          localDirectory + "/../../../testdata/bad_data/v2/",
			virtualMemoryStat: mem.VirtualMemoryStat{},
			errorType:         &strconv.NumError{},
		},
		{
			name:              "Test 7: no file",
			basePath:          localDirectory + "/unknown/",
			virtualMemoryStat: mem.VirtualMemoryStat{},
			errorType:         &os.PathError{},
		},
	}

	getHostMemoryStats = func(ctx context.Context) (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Total: 636870912}, nil
	}

	pageSize = 65536

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			cgroupCPUSource := NewMemorySource(test.basePath)
			virtualMemoryStat, err := cgroupCPUSource.VirtualMemoryStat()

			// Assert error
			if test.errorType == nil {
				require.NoError(tt, err)
			} else {
				require.Error(tt, err)
				// satisfy the linter's requirement for a more specific check than IsType.
				require.Condition(tt, func() bool {
					return errors.As(err, &test.errorType)
				}, "Error should be of type %T", test.errorType)
			}

			// Assert result
			require.Equal(tt, test.virtualMemoryStat, *virtualMemoryStat)
		})
	}
}
