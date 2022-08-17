package cgroup

import (
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPercentages(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name      string
		basePath  string
		cpuStat   DockerCpuPercentages
		errorType error
	}{
		{"v1 good data", localDirectory + "/../testdata/good_data/v1/", DockerCpuPercentages{User: 0.6712570862198262, System: 0.20429056808044366}, nil},
		{"v1 bad data", localDirectory + "/../testdata/bad_data/v1/", DockerCpuPercentages{}, &strconv.NumError{}},
		{"v2 good data", localDirectory + "/../testdata/good_data/v2/", DockerCpuPercentages{User: 4.627063395919899, System: 4.250076104937527}, nil},
		{"v2 bad data", localDirectory + "/../testdata/bad_data/v2/", DockerCpuPercentages{}, &strconv.NumError{}},
		{"no file", localDirectory + "/unknown/", DockerCpuPercentages{}, &os.PathError{}},
	}

	GetNumberOfCores = func() int {
		return 2
	}
	CpuStatsPath = localDirectory + "/../testdata/proc/stat"

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			cgroupCPUSource := NewCgroupCPUSource(test.basePath)
			cpuStat, err := cgroupCPUSource.Percentages()

			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, test.cpuStat, cpuStat)
		})
	}
}
