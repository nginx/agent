// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cgroup

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
	"log/slog"
	"path"
	"strconv"
	"strings"
)

/*
cgroup memory files (under /sys/fs/cgroup/)

memory.current
memory.events
memory.events.local
memory.high
memory.low
memory.max
memory.min
memory.numa_stat
memory.oom.group
memory.peak
memory.pressure
memory.reclaim
memory.stat
memory.swap.current
memory.swap.events
memory.swap.high
memory.swap.max
memory.swap.peak
memory.zswap.current
memory.zswap.max
memory.zswap.writeback
*/

type MemorySource struct {
	basePath   string
	isCgroupV2 bool
}

type MemoryStat struct {
	cached uint64
	shared uint64
}

var getHostMemoryStats = mem.VirtualMemoryWithContext

func NewMemorySource(basePath string) *MemorySource {
	return &MemorySource{
		basePath:   basePath,
		isCgroupV2: IsCgroupV2(basePath),
	}
}

func (ms *MemorySource) Collect() {
	_, err := ms.VirtualMemoryStatWithContext(context.Background())
	if err != nil {
		slog.Error(err.Error())
		return
	}
}
func (ms *MemorySource) VirtualMemoryStatWithContext(ctx context.Context) (*mem.VirtualMemoryStat, error) {
	var cgroupStat mem.VirtualMemoryStat
	var memoryStat MemoryStat

	// cgroup v2 by default
	memTotalFile := V2MemTotal
	memUsageFile := V2MemUsage
	memStatFile := V2MemStat
	memCachedKey := V2CachedKey
	memSharedKey := V2SharedKey
	if !ms.isCgroupV2 {
		memTotalFile = V1MemTotalFile
		memUsageFile = V1MemUsageFile
		memStatFile = V1MemStatFile
		memCachedKey = V1CachedKey
		memSharedKey = V1SharedKey
	}

	memoryLimitInBytes, err := GetMemoryLimitInBytes(path.Join(ms.basePath, memTotalFile))
	if err != nil {
		slog.Debug("Error getting memory limit in bytes", "err", err)
	}

	memoryUsageInBytes, err := ReadIntegerValueCgroupFile(path.Join(ms.basePath, memUsageFile))
	if err != nil {
		slog.Debug("Error reading memory usage in bytes", "err", err)
	}

	memoryStat, err = GetMemoryStat(
		path.Join(ms.basePath, memStatFile),
		memCachedKey,
		memSharedKey,
	)
	if err != nil {
		slog.Debug("Error getting memory stats", "err", err)
		return nil, err
	}

	var usedMemoryPercent float64

	usedMemory := memoryUsageInBytes - memoryStat.cached

	if memoryLimitInBytes > 0 {
		usedMemoryPercent = float64(100 * usedMemory / memoryLimitInBytes)
	}

	cgroupStat.Total = memoryLimitInBytes
	cgroupStat.Available = memoryLimitInBytes - usedMemory
	cgroupStat.Used = usedMemory
	cgroupStat.Cached = memoryStat.cached
	cgroupStat.Shared = memoryStat.shared
	cgroupStat.UsedPercent = usedMemoryPercent
	cgroupStat.Free = memoryLimitInBytes - usedMemory

	return &cgroupStat, nil
}

func (cs *MemorySource) VirtualMemoryStat() (*mem.VirtualMemoryStat, error) {
	ctx := context.Background()
	defer ctx.Done()
	return cs.VirtualMemoryStatWithContext(ctx)
}

func GetMemoryLimitInBytes(filePath string) (uint64, error) {
	ctx := context.Background()
	defer ctx.Done()
	memTotalString, err := ReadSingleValueCgroupFile(filePath)
	if err != nil {
		return 0, err
	}
	if memTotalString == V2DefaultMaxValue || memTotalString == GetV1DefaultMaxValue() {
		hostMemoryStats, err := getHostMemoryStats(ctx)
		if err != nil {
			return 0, nil
		}
		return hostMemoryStats.Total, nil
	} else {
		return strconv.ParseUint(memTotalString, 10, 64)
	}
}

func GetMemoryStat(statFile string, cachedKey string, sharedKey string) (MemoryStat, error) {
	memoryStat := MemoryStat{}
	lines, err := ReadLines(statFile)
	if err != nil {
		return memoryStat, err
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return memoryStat, fmt.Errorf("%+v required 2 fields", fields)
		}

		if fields[0] == cachedKey {
			value, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
			if err != nil {
				return memoryStat, err
			}
			memoryStat.cached = value
		}
		if fields[0] == sharedKey {
			value, err := strconv.ParseUint(strings.TrimSpace(fields[1]), 10, 64)
			if err != nil {
				return memoryStat, err
			}
			memoryStat.shared = value
		}
	}

	return memoryStat, nil
}
