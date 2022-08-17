package cgroup

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/mem"
)

// 02/02/2022
// Note: Buffered memory metric is not present in the memory.stat file.
// See here for more details: https://stackoverflow.com/a/52933753

var getHostMemoryStats = mem.VirtualMemory

type MemoryStat struct {
	cached uint64
	shared uint64
}

type CgroupMem struct {
	basePath   string
	isCgroupV2 bool
}

func NewCgroupMemSource(basePath string) *CgroupMem {
	return &CgroupMem{basePath, IsCgroupV2(basePath)}
}

func (cgroupMem *CgroupMem) VirtualMemoryStat() (*mem.VirtualMemoryStat, error) {
	var cgroupStat mem.VirtualMemoryStat
	var memoryLimitInBytes, memoryUsageInBytes uint64
	var memoryStat MemoryStat
	var err error

	if cgroupMem.isCgroupV2 {
		memoryLimitInBytes, err = GetMemoryLimitInBytes(path.Join(cgroupMem.basePath, V2MemTotalFile))
	} else {
		memoryLimitInBytes, err = GetMemoryLimitInBytes(path.Join(cgroupMem.basePath, V1MemTotalFile))
	}

	if err != nil {
		return &cgroupStat, err
	}

	if cgroupMem.isCgroupV2 {
		memoryUsageInBytes, err = ReadIntegerValueCgroupFile(path.Join(cgroupMem.basePath, V2MemUsageFile))
	} else {
		memoryUsageInBytes, err = ReadIntegerValueCgroupFile(path.Join(cgroupMem.basePath, V1MemUsageFile))
	}
	if err != nil {
		return &cgroupStat, err
	}

	if cgroupMem.isCgroupV2 {
		memoryStat, err = GetMemoryStat(
			path.Join(cgroupMem.basePath, V2MemStatFile),
			V2CachedKey,
			V2SharedKey,
		)
	} else {
		memoryStat, err = GetMemoryStat(
			path.Join(cgroupMem.basePath, V1MemStatFile),
			V1CachedKey,
			V1SharedKey,
		)
	}

	if err != nil {
		return &cgroupStat, err
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

func GetMemoryLimitInBytes(filePath string) (uint64, error) {
	memTotalString, err := ReadSingleValueCgroupFile(filePath)
	if err != nil {
		return 0, err
	}
	if memTotalString == V2DefaultMaxValue || memTotalString == GetV1DefaultMaxValue() {
		hostMemoryStats, err := getHostMemoryStats()
		if err != nil {
			return 0, nil
		}
		return hostMemoryStats.Total, nil
	} else {
		return strconv.ParseUint(memTotalString, 10, 64)
	}
}
