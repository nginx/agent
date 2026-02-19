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
	"github.com/stretchr/testify/require"
)

func TestCollectCPUStats(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	ctx := context.Background()

	tests := []struct {
		errorType error
		name      string
		basePath  string
		cpuStat   ContainerCPUStats
	}{
		{
			name:     "Test 1: v1 good data",
			basePath: localDirectory + "/../../../testdata/good_data/v1/",
			cpuStat: ContainerCPUStats{
				NumberOfLogicalCPUs: 2,
				User:                0.006712570862198262,
				System:              0.0020429056808044366,
			},
			errorType: nil,
		},
		{
			name:      "Test 2: v1 bad data",
			basePath:  localDirectory + "/../../../testdata/bad_data/v1/",
			cpuStat:   ContainerCPUStats{},
			errorType: &strconv.NumError{},
		},
		{
			name:     "Test 3: v2 good data",
			basePath: localDirectory + "/../../../testdata/good_data/v2/",
			cpuStat: ContainerCPUStats{
				NumberOfLogicalCPUs: 2,
				User:                0.04627063395919899,
				System:              0.04250076104937527,
			},
			errorType: nil,
		},
		{
			name:      "Test 4: v2 bad data",
			basePath:  localDirectory + "/../../../testdata/bad_data/v2/",
			cpuStat:   ContainerCPUStats{},
			errorType: &strconv.NumError{},
		},
		{
			name:      "Test 5: no file",
			basePath:  localDirectory + "/unknown/",
			cpuStat:   ContainerCPUStats{},
			errorType: &os.PathError{},
		},
	}

	GetNumberOfCores = func() int {
		return 2
	}
	CPUStatsPath = localDirectory + "/../../../testdata/proc/stat"

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			cgroupCPUSource := NewCPUSource(test.basePath)
			cpuStat, err := cgroupCPUSource.collectCPUStats(ctx)

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
			require.Equal(tt, test.cpuStat, cpuStat)
		})
	}
}
