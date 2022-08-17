package cgroup

import (
	"os"
	"path"
	"runtime"
	"strconv"
	"testing"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
)

func TestSwapMemoryStat(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name           string
		basePath       string
		swapMemoryStat mem.SwapMemoryStat
		errorType      error
	}{
		{
			"v1 good data",
			localDirectory + "/../testdata/good_data/v1/",
			mem.SwapMemoryStat{
				Total:       200000000,
				Free:        187130368,
				Used:        12869632,
				UsedPercent: 6.4348160000000005,
			},
			nil,
		},
		{
			"v1 good data no limits",
			localDirectory + "/../testdata/good_data_no_limits/v1/",
			mem.SwapMemoryStat{
				Total:       936870912,
				Free:        924001280,
				Used:        12869632,
				UsedPercent: 1.3736825250051097,
			},
			nil,
		},
		{
			"v1 bad data",
			localDirectory + "/../testdata/bad_data/v1/",
			mem.SwapMemoryStat{},
			&strconv.NumError{},
		},
		{
			"v2 good data",
			localDirectory + "/../testdata/good_data/v2/",
			mem.SwapMemoryStat{
				Total:       200000000,
				Free:        187130368,
				Used:        12869632,
				UsedPercent: 6.4348160000000005,
			},
			nil,
		},
		{
			"v2 good data no limits",
			localDirectory + "/../testdata/good_data_no_limits/v2/",
			mem.SwapMemoryStat{
				Total:       936870912,
				Free:        924001280,
				Used:        12869632,
				UsedPercent: 1.3736825250051097,
			},
			nil,
		},
		{
			"v2 bad data",
			localDirectory + "/../testdata/bad_data/v2/",
			mem.SwapMemoryStat{},
			&strconv.NumError{},
		},
		{
			"no file",
			localDirectory + "/unknown/",
			mem.SwapMemoryStat{},
			&os.PathError{},
		},
	}

	getHostSwapStats = func() (*mem.SwapMemoryStat, error) {
		return &mem.SwapMemoryStat{Total: 936870912}, nil
	}

	pageSize = 65536

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			cgroupSwapSource := NewCgroupSwapSource(test.basePath)
			swapMemoryStat, err := cgroupSwapSource.SwapMemoryStat()

			// Assert error
			assert.IsType(tt, test.errorType, err)

			// Assert result
			assert.Equal(tt, test.swapMemoryStat, *swapMemoryStat)
		})
	}
}
