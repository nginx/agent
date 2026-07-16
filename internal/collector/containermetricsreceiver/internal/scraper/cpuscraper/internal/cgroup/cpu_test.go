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
	"github.com/stretchr/testify/assert"
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

			if test.errorType != nil {
				// satisfy the linter's requirement for a more specific check than IsType.
				require.Condition(tt, func() bool {
					return errors.As(err, &test.errorType)
				}, "Error should be of type %T", test.errorType)
			} else {
				require.NoError(tt, err)
			}

			// Assert result
			assert.Equal(tt, test.cpuStat, cpuStat)
		})
	}
}

func TestCpuUsageTimes_BlankLine(t *testing.T) {
	cs := &CPUSource{}

	tests := []struct {
		name     string
		content  string
		userKey  string
		sysKey   string
		wantUser float64
		wantSys  float64
	}{
		{
			name:     "v1: blank lines interspersed are skipped",
			content:  "\nuser 5760\n\nsystem 1753\n",
			userKey:  V1UserKey,
			sysKey:   V1SystemKey,
			wantUser: 5760,
			wantSys:  1753,
		},
		{
			name:     "v2: blank lines interspersed are skipped",
			content:  "\nuser_usec 397044377\n\nsystem_usec 364695418\n",
			userKey:  V2UserKey,
			sysKey:   V2SystemKey,
			wantUser: 397044377,
			wantSys:  364695418,
		},
		{
			name:     "key-only line (no value) is skipped without panic",
			content:  "user\nsystem 1753\n",
			userKey:  V1UserKey,
			sysKey:   V1SystemKey,
			wantUser: 0,
			wantSys:  1753,
		},
		{
			name:     "empty file returns zero values without error",
			content:  "",
			userKey:  V1UserKey,
			sysKey:   V1SystemKey,
			wantUser: 0,
			wantSys:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "cgroup-stat-*")
			require.NoError(t, err)
			_, err = f.WriteString(test.content)
			require.NoError(t, err)
			require.NoError(t, f.Close())

			cpuTimes, err := cs.cpuUsageTimes(f.Name(), test.userKey, test.sysKey)
			require.NoError(t, err)
			assert.InDelta(t, test.wantUser, cpuTimes.userUsage, 0.001)
			assert.InDelta(t, test.wantSys, cpuTimes.systemUsage, 0.001)
		})
	}
}

// Tests that a blank line in /proc/stat is skipped without an index-out-of-range panic.
func TestSystemCPUUsage_BlankLine(t *testing.T) {
	// proc/stat with a leading blank line followed by the normal cpu line.
	content := "\ncpu  366663 264 272326 1072402 2744 0 1784 0 0 0\n"

	f, err := os.CreateTemp(t.TempDir(), "proc-stat-*")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	original := CPUStatsPath
	CPUStatsPath = f.Name()
	defer func() { CPUStatsPath = original }()

	result, err := systemCPUUsage(100)
	require.NoError(t, err)
	assert.Greater(t, result, float64(0), "expected non-zero CPU usage from a valid cpu line")
}
