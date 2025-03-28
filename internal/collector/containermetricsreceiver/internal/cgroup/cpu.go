// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package cgroup

// nolint: unused
const (
	nanoSecondsPerSecond = 1e9
)

// nolint: unused
type DockerCPUTimes struct {
	userUsage       float64
	systemUsage     float64
	hostSystemUsage float64
}

// nolint: unused
type DockerCPUPercentages struct {
	User   float64
	System float64
}

// nolint: unused
type CgroupCPUSource struct {
	basePath   string
	isCgroupV2 bool
	clockTicks int
}

func NewCgroupCPUSource() *CgroupCPUSource {
	return &CgroupCPUSource{}
}
