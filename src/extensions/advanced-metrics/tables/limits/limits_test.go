/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package limits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitsCollapsingLevel(t *testing.T) {
	tests := []struct {
		Name          string
		Max           int
		Threshold     int
		CurrentSize   int
		ExpectedLevel CollapsingLevel
		ExpectedError string
	}{
		{
			Name:          "error when threshold greater than max",
			Max:           1,
			Threshold:     10,
			CurrentSize:   0,
			ExpectedLevel: 0,
			ExpectedError: "threshold must be less than maxCapacity",
		},
		{
			Name:          "error when max equal 0",
			Max:           0,
			Threshold:     0,
			CurrentSize:   0,
			ExpectedLevel: 0,
			ExpectedError: "maxCapacity must be greater than 0",
		},
		{
			Name:          "collapsing level 0 when current size bellow threshold",
			Max:           100,
			Threshold:     50,
			CurrentSize:   10,
			ExpectedLevel: 0,
		},
		{
			Name:          "collapsing level 0 when current size equal threshold",
			Max:           100,
			Threshold:     50,
			CurrentSize:   50,
			ExpectedLevel: 0,
		},
		{
			Name:          "collapsing level calculated properly when current size between max and threshold",
			Max:           100,
			Threshold:     50,
			CurrentSize:   60,
			ExpectedLevel: 100 * 10. / 50.,
		},
		{
			Name:          "collapsing level calculated properly when current size between max and threshold",
			Max:           100,
			Threshold:     50,
			CurrentSize:   90,
			ExpectedLevel: 100 * 40. / 50.,
		},
		{
			Name:          "collapsing level calculated properly when current size equals max",
			Max:           100,
			Threshold:     50,
			CurrentSize:   100,
			ExpectedLevel: 100 * 50. / 50.,
		},
		{
			Name:          "collapsing level is equal 100 when current size is greater than max",
			Max:           100,
			Threshold:     50,
			CurrentSize:   120,
			ExpectedLevel: MaxCollapseLevel,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			limit, err := NewLimits(test.Max, test.Threshold)
			if test.ExpectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.ExpectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.ExpectedLevel, limit.GetCurrentCollapsingLevel(test.CurrentSize))
			}
		})

	}

}
