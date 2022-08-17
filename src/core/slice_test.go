package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceContainsString(t *testing.T) {
	testCases := []struct {
		testName     string
		sliceToCheck []string
		stringToFind string
		expFound     bool
		expIndex     int
	}{
		{
			testName:     "StringFoundInSlice",
			sliceToCheck: []string{"slice", "to", "check"},
			stringToFind: "check",
			expFound:     true,
			expIndex:     2,
		},
		{
			testName:     "StringNotFoundInSlice",
			sliceToCheck: []string{"slice", "to", "check"},
			stringToFind: "not in slice",
			expFound:     false,
			expIndex:     -1,
		},
		{
			testName:     "EmptySlice",
			sliceToCheck: []string{},
			stringToFind: "nothing",
			expFound:     false,
			expIndex:     -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			found, index := SliceContainsString(tc.sliceToCheck, tc.stringToFind)
			assert.Equal(t, tc.expFound, found)
			assert.Equal(t, tc.expIndex, index)
		})
	}
}
