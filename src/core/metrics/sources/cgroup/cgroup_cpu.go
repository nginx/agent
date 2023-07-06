/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package cgroup

import (
	"bytes"
	"errors"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

// 02/02/2022
// Note: Only user & system metrics are returned. Other metrics like iowait are left empty.
// See here for more details: https://www.datadoghq.com/blog/how-to-monitor-docker-resource-metrics/#standard-metrics

const nanoSecondsPerSecond = 1e9

var (
	CpuStatsPath     = "/proc/stat"
	GetNumberOfCores = runtime.NumCPU
)

type DockerCpuTimes struct {
	userUsage          float64
	systemUsage        float64
	hostSystemCpuUsage float64
}

type DockerCpuPercentages struct {
	User   float64
	System float64
}

type CgroupCPU struct {
	basePath               string
	isCgroupV2             bool
	previousDockerCpuTimes *DockerCpuTimes
	clockTicks             int
}

func NewCgroupCPUSource(basePath string) *CgroupCPU {
	clockTicks, err := getClockTicks()
	if err != nil {
		log.Errorf("Failed to create CgroupCPU: %v", err)
		return nil
	}
	return &CgroupCPU{basePath, IsCgroupV2(basePath), &DockerCpuTimes{userUsage: 0, systemUsage: 0, hostSystemCpuUsage: 0}, clockTicks}
}

func (cgroupCPU *CgroupCPU) Percentages() (DockerCpuPercentages, error) {
	var dockerCpuTimes *DockerCpuTimes
	var err error

	if cgroupCPU.isCgroupV2 {
		dockerCpuTimes, err = getCPUStat(
			path.Join(cgroupCPU.basePath, V2CpuStatFile),
			V2UserKey,
			V2SystemKey,
		)

		// CPU times are in microseconds and get converted to nanoseconds
		dockerCpuTimes.userUsage = dockerCpuTimes.userUsage * 1000
		dockerCpuTimes.systemUsage = dockerCpuTimes.systemUsage * 1000
	} else {
		dockerCpuTimes, err = getCPUStat(
			path.Join(cgroupCPU.basePath, V1CpuacctStatFile),
			V1UserKey,
			V1SystemKey,
		)

		// CPU times are in USER_HZ and get converted to nanoseconds
		dockerCpuTimes.userUsage = (dockerCpuTimes.userUsage * nanoSecondsPerSecond) / float64(cgroupCPU.clockTicks)
		dockerCpuTimes.systemUsage = (dockerCpuTimes.systemUsage * nanoSecondsPerSecond) / float64(cgroupCPU.clockTicks)
	}

	if err != nil {
		return DockerCpuPercentages{}, err
	}

	hostSystemCpuUsage, err := getSystemCPUUsage(cgroupCPU.clockTicks)
	if err != nil {
		return DockerCpuPercentages{}, err
	}

	dockerCpuTimes.hostSystemCpuUsage = hostSystemCpuUsage

	log.Tracef("Previous Docker CPU Times: %v", cgroupCPU.previousDockerCpuTimes)
	log.Tracef("Current Docker CPU Times: %v", dockerCpuTimes)

	userDelta := dockerCpuTimes.userUsage - cgroupCPU.previousDockerCpuTimes.userUsage
	systemDelta := dockerCpuTimes.systemUsage - cgroupCPU.previousDockerCpuTimes.systemUsage
	hostSystemDelta := dockerCpuTimes.hostSystemCpuUsage - cgroupCPU.previousDockerCpuTimes.hostSystemCpuUsage

	log.Tracef("User CPU Delta: %f", userDelta)
	log.Tracef("System CPU Delta: %f", systemDelta)
	log.Tracef("Host System CPU Delta: %f", hostSystemDelta)

	cpuCores := GetNumberOfCores()
	log.Tracef("Number of CPU cores: %d", cpuCores)

	userPercentage := ((userDelta / hostSystemDelta) * float64(cpuCores) * 100)
	systemPercentage := ((systemDelta / hostSystemDelta) * float64(cpuCores) * 100)

	log.Tracef("User CPU Percentage: %f", userPercentage)
	log.Tracef("System CPU Percentage: %f", systemPercentage)

	dockerCpuPercentages := DockerCpuPercentages{
		User:   userPercentage,
		System: systemPercentage,
	}

	log.Tracef("Docker CPU Percentages: %v", dockerCpuPercentages)

	cgroupCPU.previousDockerCpuTimes = dockerCpuTimes

	return dockerCpuPercentages, nil
}

func getSystemCPUUsage(clockTicks int) (float64, error) {
	lines, err := ReadLines(CpuStatsPath)
	if err != nil {
		return 0, err
	}

	for _, line := range lines {
		parts := strings.Fields(line)
		switch parts[0] {
		case "cpu":
			if len(parts) < 8 {
				return 0, errors.New("unable to process " + CpuStatsPath + ". Invalid number of fields for cpu line")
			}
			var totalClockTicks float64
			for _, i := range parts[1:8] {
				v, err := strconv.ParseFloat(i, 64)
				if err != nil {
					return 0, err
				}
				totalClockTicks += v
			}

			return (totalClockTicks * nanoSecondsPerSecond) / float64(clockTicks), nil
		}
	}

	return 0, errors.New("unable to process " + CpuStatsPath + ". No cpu found")
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

func getCPUStat(statFile string, user_key string, system_key string) (*DockerCpuTimes, error) {
	ret := &DockerCpuTimes{}

	lines, err := ReadLines(statFile)
	if err != nil {
		return ret, err
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == user_key {
			user, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return ret, err
			}

			ret.userUsage = user
		}
		if fields[0] == system_key {
			system, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return ret, err
			}

			ret.systemUsage = system
		}
	}

	return ret, nil
}
