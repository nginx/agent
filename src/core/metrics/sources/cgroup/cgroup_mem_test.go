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
	"strconv"
	"testing"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
)

func TestVirtualMemoryStat(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name              string
		basePath          string
		virtualMemoryStat mem.VirtualMemoryStat
		errorType         error
	}{
		{
			"v1 good data",
			localDirectory + "/../testdata/good_data/v1/",
			mem.VirtualMemoryStat{
				Total:       536870912,
				Free:        420200448,
				Available:   420200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 21,
			},
			nil,
		},
		{
			"v1 good data no limits",
			localDirectory + "/../testdata/good_data_no_limits/v1/",
			mem.VirtualMemoryStat{
				Total:       636870912,
				Free:        520200448,
				Available:   520200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 18,
			},
			nil,
		},
		{
			"v1 bad data",
			localDirectory + "/../testdata/bad_data/v1/",
			mem.VirtualMemoryStat{},
			&strconv.NumError{},
		},
		{
			"v2 good data",
			localDirectory + "/../testdata/good_data/v2/",
			mem.VirtualMemoryStat{
				Total:       536870912,
				Free:        420200448,
				Available:   420200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 21,
			},
			nil,
		},
		{
			"v2 good data no limits",
			localDirectory + "/../testdata/good_data_no_limits/v2/",
			mem.VirtualMemoryStat{
				Total:       636870912,
				Free:        520200448,
				Available:   520200448,
				Used:        116670464,
				Cached:      275480576,
				Shared:      53805056,
				UsedPercent: 18,
			},
			nil,
		},
		{
			"v2 bad data",
			localDirectory + "/../testdata/bad_data/v2/",
			mem.VirtualMemoryStat{},
			&strconv.NumError{},
		},
		{
			"no file",
			localDirectory + "/unknown/",
			mem.VirtualMemoryStat{},
			&os.PathError{},
		},
	}

	getHostMemoryStats = func() (*mem.VirtualMemoryStat, error) {
		return &mem.VirtualMemoryStat{Total: 636870912}, nil
	}

	pageSize = 65536

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			cgroupCPUSource := NewCgroupMemSource(test.basePath)
			virtualMemoryStat, err := cgroupCPUSource.VirtualMemoryStat()

			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, test.virtualMemoryStat, *virtualMemoryStat)
		})
	}
}
