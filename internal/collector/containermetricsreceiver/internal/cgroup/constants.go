// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cgroup

const (
	CgroupBasePath = "/sys/fs/cgroup/"

	V1CpuacctStatFile        = CgroupBasePath + "cpuacct.stat"
	V1CpuStatFile            = CgroupBasePath + "cpu.stat"
	V1CpuSharesFile          = CgroupBasePath + "cpu.shares"
	V1CpuPeriodFile          = CgroupBasePath + "cpu.cfs_period_us"
	V1CpuQuotaFile           = CgroupBasePath + "cpu.cfs_quota_us"
	V1CpusetCpusFile         = CgroupBasePath + "cpuset.cpus"
	V1MemStatFile            = CgroupBasePath + "memory.stat"
	V1MemTotalFile           = CgroupBasePath + "memory.limit_in_bytes"
	V1MemUsageFile           = CgroupBasePath + "memory.usage_in_bytes"
	V1OutOfMemoryControlFile = CgroupBasePath + "memory.oom_control"
	V1SwapTotalFile          = CgroupBasePath + "memory.memsw.limit_in_bytes"
	V1SwapUsageFile          = CgroupBasePath + "memory.memsw.usage_in_bytes"

	V1UserKey                = "user"
	V1SystemKey              = "system"
	V1CachedKey              = "cache"
	V1SharedKey              = "total_shmem"
	V1ThrottlingTimeKey      = "throttled_time"
	V1ThrottlingThrottledKey = "nr_throttled"
	V1ThrottlingPeriodsKey   = "nr_periods"
	V1OutOfMemoryKey         = "under_oom"
	V1OutOfMemoryKillKey     = "oom_kill"

	V2CpuStatFile    = CgroupBasePath + "cpu.stat"
	V2CpuWeightFile  = CgroupBasePath + "cpu.weight"
	V2CpuMaxFile     = CgroupBasePath + "cpu.max"
	V2CpusetCpusFile = CgroupBasePath + "cpuset.cpus"
	V2MemStatFile    = CgroupBasePath + "memory.stat"
	V2MemTotalFile   = CgroupBasePath + "memory.max"
	V2MemUsageFile   = CgroupBasePath + "memory.current"
	V2MemEventsFile  = CgroupBasePath + "memory.events"
	V2SwapTotalFile  = CgroupBasePath + "memory.swap.max"
	V2SwapUsageFile  = CgroupBasePath + "memory.swap.current"

	V2UserKey                = "user_usec"
	V2SystemKey              = "system_usec"
	V2CachedKey              = "file"
	V2SharedKey              = "shmem"
	V2ThrottlingTimeKey      = "throttled_usec"
	V2ThrottlingThrottledKey = "nr_throttled"
	V2ThrottlingPeriodsKey   = "nr_periods"
	V2OutOfMemoryKey         = "oom"
	V2OutOfMemoryKillKey     = "oom_kill"
	V2DefaultMaxValue        = "max"
)
