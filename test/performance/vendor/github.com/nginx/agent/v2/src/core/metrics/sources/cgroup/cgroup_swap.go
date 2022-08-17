package cgroup

import (
	"path"
	"strconv"

	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

var getHostSwapStats = mem.SwapMemory

type swap struct {
	memTotal, memUsage,
	total, used uint64
}

func (s *swap) Total() uint64 {
	if s.total < s.memTotal {
		return 0
	}
	return s.total - s.memTotal
}

func (s *swap) Free() uint64 {
	if s.Total() < s.Used() {
		return 0
	}
	return s.Total() - s.Used()
}

func (s *swap) Used() uint64 {
	if s.used < s.memUsage {
		return 0
	}
	return s.used - s.memUsage
}

func (s *swap) UsedPercent() float64 {
	if s.Total() == 0 {
		return 0
	}
	return float64(100) * (float64(s.Used()) / float64(s.Total()))
}

type CgroupSwap struct {
	basePath   string
	isCgroupV2 bool
}

func NewCgroupSwapSource(basePath string) *CgroupSwap {
	return &CgroupSwap{basePath, IsCgroupV2(basePath)}
}

func (cgroupSwap *CgroupSwap) SwapMemoryStat() (*mem.SwapMemoryStat, error) {
	cgroupStat := &mem.SwapMemoryStat{}
	var memTotal, memUsage, total, used uint64
	var err error

	if cgroupSwap.isCgroupV2 {
		memTotal, err = GetMemTotal(path.Join(cgroupSwap.basePath, V2MemTotalFile))
	} else {
		memTotal, err = GetMemTotal(path.Join(cgroupSwap.basePath, V1MemTotalFile))
	}
	if err != nil {
		return cgroupStat, err
	}

	if cgroupSwap.isCgroupV2 {
		memUsage, err = ReadIntegerValueCgroupFile(path.Join(cgroupSwap.basePath, V2MemUsageFile))
	} else {
		memUsage, err = ReadIntegerValueCgroupFile(path.Join(cgroupSwap.basePath, V1MemUsageFile))
	}
	if err != nil {
		return cgroupStat, err
	}

	if cgroupSwap.isCgroupV2 {
		total, err = GetTotal(path.Join(cgroupSwap.basePath, V2SwapTotalFile))
	} else {
		total, err = GetTotal(path.Join(cgroupSwap.basePath, V1SwapTotalFile))
	}
	if err != nil {
		return cgroupStat, err
	}

	if cgroupSwap.isCgroupV2 {
		used, err = ReadIntegerValueCgroupFile(path.Join(cgroupSwap.basePath, V2SwapUsageFile))
	} else {
		used, err = ReadIntegerValueCgroupFile(path.Join(cgroupSwap.basePath, V1SwapUsageFile))
	}
	if err != nil {
		return cgroupStat, err
	}

	swapStat := &swap{memTotal, memUsage, total, used}

	if swapStat.Total() != 0 {
		cgroupStat.Total = swapStat.Total()
		cgroupStat.Free = swapStat.Free()
		cgroupStat.Used = swapStat.Used()
		cgroupStat.UsedPercent = swapStat.UsedPercent()
	}

	return cgroupStat, nil
}

// If no memory limit is set for docker container,
// then memTotal is set to zero and total is set to the host's swap memory total
func GetMemTotal(filePath string) (uint64, error) {
	memTotalString, err := ReadSingleValueCgroupFile(filePath)
	if err != nil {
		return 0, err
	}
	if memTotalString == V2DefaultMaxValue || memTotalString == GetV1DefaultMaxValue() {
		return 0, nil
	} else {
		return strconv.ParseUint(memTotalString, 10, 64)
	}
}

func GetTotal(filePath string) (uint64, error) {
	totalString, err := ReadSingleValueCgroupFile(filePath)
	if err != nil {
		return 0, err
	}

	hostSwapStats, err := getHostSwapStats()
	if err != nil {
		return 0, err
	}

	if totalString == V2DefaultMaxValue || totalString == GetV1DefaultMaxValue() {
		return hostSwapStats.Total, nil
	} else {
		swapTotal, err := strconv.ParseUint(totalString, 10, 64)
		if err != nil {
			return 0, err
		}

		// If the host system has less swap memory allocated then the swap memory set for the docker container
		// we will only display the actual amount of swap memory that the container has.
		if hostSwapStats.Total < swapTotal {
			log.Warnf(
				"Swap memory limit specified for the container, %d is greater than the host system swap memory, %d",
				swapTotal,
				hostSwapStats.Total,
			)
			return hostSwapStats.Total, nil
		}

		return swapTotal, nil
	}
}
