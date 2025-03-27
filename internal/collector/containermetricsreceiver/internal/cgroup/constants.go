/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cgroup

const (
	CgroupBasePath = "/sys/fs/cgroup/"

	V1CpuacctStatFile        = "cpuacct/cpuacct.stat"
	V1CpuStatFile            = "cpu/cpu.stat"
	V1CpuSharesFile          = "cpu/cpu.shares"
	V1CpuPeriodFile          = "cpu/cpu.cfs_period_us"
	V1CpuQuotaFile           = "cpu/cpu.cfs_quota_us"
	V1CpusetCpusFile         = "cpuset/cpuset.cpus"
	V1MemStatFile            = "memory/memory.stat"
	V1MemTotalFile           = "memory/memory.limit_in_bytes"
	V1MemUsageFile           = "memory/memory.usage_in_bytes"
	V1OutOfMemoryControlFile = "memory/memory.oom_control"
	V1SwapTotalFile          = "memory/memory.memsw.limit_in_bytes"
	V1SwapUsageFile          = "memory/memory.memsw.usage_in_bytes"

	V1UserKey                = "user"
	V1SystemKey              = "system"
	V1CachedKey              = "cache"
	V1SharedKey              = "total_shmem"
	V1ThrottlingTimeKey      = "throttled_time"
	V1ThrottlingThrottledKey = "nr_throttled"
	V1ThrottlingPeriodsKey   = "nr_periods"
	V1OutOfMemoryKey         = "under_oom"
	V1OutOfMemoryKillKey     = "oom_kill"

	V2CpuStatFile    = "cpu.stat"
	V2CpuWeightFile  = "cpu.weight"
	V2CpuMaxFile     = "cpu.max"
	V2CpusetCpusFile = "cpuset.cpus"
	V2MemStatFile    = "memory.stat"
	V2MemTotalFile   = "memory.max"
	V2MemUsageFile   = "memory.current"
	V2MemEventsFile  = "memory.events"
	V2SwapTotalFile  = "memory.swap.max"
	V2SwapUsageFile  = "memory.swap.current"

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
