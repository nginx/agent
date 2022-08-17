package lookup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupBytes(t *testing.T) {
	tests := []struct {
		name          string
		size          uint32
		dataToLookup  []string
		expectedCodes []int
	}{
		{
			name: "lookup single value",
			size: 10,
			dataToLookup: []string{
				"data1",
			},
			expectedCodes: []int{
				2,
			},
		},
		{
			name: "lookup multiple values",
			size: 10,
			dataToLookup: []string{
				"data1",
				"data2",
				"data3",
				"data4",
			},
			expectedCodes: []int{
				2,
				3,
				4,
				5,
			},
		},
		{
			name: "lookup multiple value with repetitions",
			size: 10,
			dataToLookup: []string{
				"data1",
				"data2",
				"data1",
				"data2",
				"data1",
				"data3",
			},
			expectedCodes: []int{
				2,
				3,
				2,
				3,
				2,
				4,
			},
		},
		{
			name: "lookup exceeding size returns aggregated code",
			size: 5,
			dataToLookup: []string{
				"data1",
				"data2",
				"data1",
				"data2",
				"data3",
				"data4",
				"data4",
				"data3",
			},
			expectedCodes: []int{
				2,
				3,
				2,
				3,
				4,
				lookupAggrCode,
				lookupAggrCode,
				4,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookup := newLookup("test", test.size)
			assert.Equal(t, len(test.dataToLookup), len(test.expectedCodes))
			for i, data := range test.dataToLookup {
				assert.Equal(t, test.expectedCodes[i], lookup.LookupBytes([]byte(data)))
			}
		})
	}
}

func TestLookupCode(t *testing.T) {
	lookup := newLookup("test", 10)

	val, err := lookup.LookupCode(lookupAggrCode)
	assert.NoError(t, err)
	assert.Equal(t, val, lookupAggr)

	data := "data1"
	code := lookup.LookupBytes([]byte(data))
	val, err = lookup.LookupCode(code)
	assert.NoError(t, err)
	assert.Equal(t, data, val)

	data = "data2"
	code = lookup.LookupBytes([]byte(data))
	val, err = lookup.LookupCode(code)
	assert.NoError(t, err)
	assert.Equal(t, data, val)
}

func TestLookupShouldFailOnUnknowCode(t *testing.T) {
	lookup := newLookup("test", 10)

	_, err := lookup.LookupCode(lookupAggrCode + 1)
	assert.Error(t, err)

}
