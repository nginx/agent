// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cgroup

import (
	"bytes"
	"errors"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal"
)

const (
	V1CpuacctStatFile = "cpuacct/cpuacct.stat"
	V1UserKey         = "user"
	V1SystemKey       = "system"

	V2CpuStat   = "cpu.stat"
	V2UserKey   = "user_usec"
	V2SystemKey = "system_usec"

	CPUStatsFileLineLength = 8
	nanoSecondsPerSecond   = 1e9
)

var (
	CPUStatsPath     = "/proc/stat"
	GetNumberOfCores = runtime.NumCPU
)

type (
	ContainerCPUTimes struct {
		userUsage       float64
		systemUsage     float64
		hostSystemUsage float64
	}

	ContainerCPUStats struct {
		NumberOfLogicalCPUs int
		User                float64
		System              float64
	}

	CPUSource struct {
		previous   *ContainerCPUTimes
		basePath   string
		isCgroupV2 bool
	}
)

func NewCPUSource(basePath string) *CPUSource {
	return &CPUSource{
		basePath:   basePath,
		isCgroupV2: internal.IsCgroupV2(basePath),
		previous:   &ContainerCPUTimes{},
	}
}

func (cs *CPUSource) Collect() (ContainerCPUStats, error) {
	cpuStats, err := cs.collectCPUStats()
	if err != nil {
		return ContainerCPUStats{}, err
	}

	return cpuStats, nil
}

// nolint: mnd
func (cs *CPUSource) collectCPUStats() (ContainerCPUStats, error) {
	clockTicks, err := getClockTicks()
	if err != nil {
		return ContainerCPUStats{}, err
	}

	// cgroup v2 by default
	filepath := path.Join(cs.basePath, V2CpuStat)
	userKey := V2UserKey
	sysKey := V2SystemKey
	convertUsage := func(usage float64) float64 {
		return usage * 1000
	}

	if !cs.isCgroupV2 { // cgroup v1
		filepath = path.Join(cs.basePath, V1CpuacctStatFile)
		userKey = V1UserKey
		sysKey = V1SystemKey
		convertUsage = func(usage float64) float64 {
			return usage * nanoSecondsPerSecond / float64(clockTicks)
		}
	}

	cpuTimes, err := cs.cpuUsageTimes(
		filepath,
		userKey,
		sysKey,
	)
	if err != nil {
		return ContainerCPUStats{}, err
	}

	cpuTimes.userUsage = convertUsage(cpuTimes.userUsage)
	cpuTimes.systemUsage = convertUsage(cpuTimes.systemUsage)
	hostSystemUsage, err := getSystemCPUUsage(clockTicks)
	if err != nil {
		return ContainerCPUStats{}, err
	}
	cpuTimes.hostSystemUsage = hostSystemUsage

	// calculate deltas
	userDelta := cpuTimes.userUsage - cs.previous.userUsage
	systemDelta := cpuTimes.systemUsage - cs.previous.systemUsage
	hostSystemDelta := cpuTimes.hostSystemUsage - cs.previous.hostSystemUsage

	numCores := GetNumberOfCores()
	userPercent := (userDelta / hostSystemDelta) * float64(numCores)
	systemPercent := (systemDelta / hostSystemDelta) * float64(numCores)

	cpuStats := ContainerCPUStats{
		NumberOfLogicalCPUs: numCores,
		User:                userPercent,
		System:              systemPercent,
	}

	cs.previous = cpuTimes

	return cpuStats, nil
}

func (cs *CPUSource) cpuUsageTimes(filePath, userKey, systemKey string) (*ContainerCPUTimes, error) {
	cpuTimes := &ContainerCPUTimes{}
	lines, err := internal.ReadLines(filePath)
	if err != nil {
		return cpuTimes, err
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		switch fields[0] {
		case userKey:
			user, parseErr := strconv.ParseFloat(fields[1], 64)
			if parseErr != nil {
				return cpuTimes, parseErr
			}
			cpuTimes.userUsage = user
		case systemKey:
			system, parseErr := strconv.ParseFloat(fields[1], 64)
			if parseErr != nil {
				return cpuTimes, parseErr
			}
			cpuTimes.systemUsage = system
		}
	}

	return cpuTimes, nil
}

// nolint: revive, gocritic
func getSystemCPUUsage(clockTicks int) (float64, error) {
	lines, err := internal.ReadLines(CPUStatsPath)
	if err != nil {
		return 0, err
	}

	for _, line := range lines {
		parts := strings.Fields(line)
		switch parts[0] {
		case "cpu":
			if len(parts) < CPUStatsFileLineLength {
				return 0, errors.New("unable to process " + CPUStatsPath + ". Invalid number of fields for cpu line")
			}
			var totalClockTicks float64
			for _, i := range parts[1:CPUStatsFileLineLength] {
				v, parseErr := strconv.ParseFloat(i, 64)
				if parseErr != nil {
					return 0, err
				}
				totalClockTicks += v
			}

			return (totalClockTicks * nanoSecondsPerSecond) / float64(clockTicks), nil
		}
	}

	return 0, errors.New("unable to process " + CPUStatsPath + ". No cpu found")
}

func getClockTicks() (int, error) {
	cmd := exec.Command("getconf", "CLK_TCK")
	out := new(bytes.Buffer)
	cmd.Stdout = out

	err := cmd.Run()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(strings.TrimSuffix(out.String(), "\n"))
}
