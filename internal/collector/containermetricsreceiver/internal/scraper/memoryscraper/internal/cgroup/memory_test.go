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
	"github.com/stretchr/testify/assert"
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
			if test.errorType != nil {
				// satisfy the linter's requirement for a more specific check than IsType.
				require.Condition(tt, func() bool {
					return errors.As(err, &test.errorType)
				}, "Error should be of type %T", test.errorType)
			} else {
				require.NoError(tt, err)
			}

			// Assert result
			assert.Equal(tt, test.virtualMemoryStat, *virtualMemoryStat)
		})
	}
}

func TestSaturatingSub(t *testing.T) {
	tests := []struct {
		name     string
		a, b     uint64
		expected uint64
	}{
		{
			name:     "normal subtraction",
			a:        100,
			b:        40,
			expected: 60,
		},
		{
			name:     "equal values return zero",
			a:        100,
			b:        100,
			expected: 0,
		},
		{
			name:     "b greater than a clamps to zero (cached exceeds usage)",
			a:        100,
			b:        200,
			expected: 0,
		},
		{
			name:     "b greater than a clamps to zero (used exceeds limit)",
			a:        400,
			b:        500,
			expected: 0,
		},
		{
			name:     "zero minus zero returns zero",
			a:        0,
			b:        0,
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, saturatingSub(test.a, test.b))
		})
	}
}

func TestVirtualMemoryStat_CachedExceedsUsage(t *testing.T) {
	dir := t.TempDir()

	// cgroup.controllers marks basePath as v2
	require.NoError(t, os.WriteFile(path.Join(dir, "cgroup.controllers"), []byte(""), 0o600))
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.max"), []byte("1000\n"), 0o600))
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.current"), []byte("100\n"), 0o600))
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.stat"), []byte("file 200\nshmem 0\n"), 0o600))

	src := NewMemorySource(dir)
	stat, err := src.VirtualMemoryStatWithContext(t.Context())
	require.NoError(t, err)

	// usedMemory = saturatingSub(100, 200) = 0 — must not wrap to 2^64
	assert.Equal(t, uint64(0), stat.Used, "Used must clamp to 0 when cached > usage")
	// Free = saturatingSub(1000, 0) = 1000 — no secondary underflow
	assert.Equal(t, uint64(1000), stat.Free, "Free must equal limit when usedMemory is 0")
	assert.InDelta(t, float64(0), stat.UsedPercent, 0.001, "UsedPercent must be 0")
}

func TestVirtualMemoryStat_UsedExceedsLimit(t *testing.T) {
	dir := t.TempDir()

	// cgroup.controllers marks basePath as v2
	require.NoError(t, os.WriteFile(path.Join(dir, "cgroup.controllers"), []byte(""), 0o600))
	// usage (500) > limit (400) — OOM/transient cgroup state
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.max"), []byte("400\n"), 0o600))
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.current"), []byte("500\n"), 0o600))
	require.NoError(t, os.WriteFile(path.Join(dir, "memory.stat"), []byte("file 0\nshmem 0\n"), 0o600))

	src := NewMemorySource(dir)
	stat, err := src.VirtualMemoryStatWithContext(t.Context())
	require.NoError(t, err)

	// usedMemory = saturatingSub(500, 0) = 500 — no underflow
	assert.Equal(t, uint64(500), stat.Used, "Used = usage - cached = 500")
	// Free = saturatingSub(400, 500) = 0 — must not wrap to 2^64
	assert.Equal(t, uint64(0), stat.Free, "Free must clamp to 0 when used > limit")
	// Available = saturatingSub(400, 500) = 0 — same expression
	assert.Equal(t, uint64(0), stat.Available, "Available must clamp to 0 when used > limit")
}
