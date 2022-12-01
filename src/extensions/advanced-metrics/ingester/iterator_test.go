/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ingester

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIterator(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		expectedFields []string
	}{
		{
			name:           "no fields",
			data:           "",
			expectedFields: []string{},
		},
		{
			name: "two empty fields",
			data: " ",
			expectedFields: []string{
				"",
				"",
			},
		},
		{
			name: "single field",
			data: "field1",
			expectedFields: []string{
				"field1",
			},
		},
		{
			name: "multiple fields",
			data: "field1 field2",
			expectedFields: []string{
				"field1",
				"field2",
			},
		},
		{
			name: "multiple fields with empty field",
			data: "field1  field2",
			expectedFields: []string{
				"field1",
				"",
				"field2",
			},
		},
		{
			name: "multiple fields with empty fields",
			data: "field1   field2",
			expectedFields: []string{
				"field1",
				"",
				"",
				"field2",
			},
		},
		{
			name: "multiple fields with empty field on begining",
			data: " field1 field2",
			expectedFields: []string{
				"",
				"field1",
				"field2",
			},
		},
		{
			name: "multiple fields with empty field on end",
			data: "field1 field2 ",
			expectedFields: []string{
				"field1",
				"field2",
				"",
			},
		},
		{
			name: "multiple fields with multiple empty field",
			data: "  field1  field2  ",
			expectedFields: []string{
				"",
				"",
				"field1",
				"",
				"field2",
				"",
				"",
			},
		},
		{
			name: "single string field with separator inside",
			data: `"asd a"`,
			expectedFields: []string{
				`"asd a"`,
			},
		},
		{
			name: "string field with separator inside",
			data: `"asd a"  field1   "asd b" field2  "asd c" `,
			expectedFields: []string{
				`"asd a"`,
				"",
				"field1",
				"",
				"",
				`"asd b"`,
				"field2",
				"",
				`"asd c"`,
				"",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			it := newMessageFieldIterator([]byte(test.data))
			for _, expectedField := range test.expectedFields {
				assert.True(t, it.HasNext())
				field := it.Next()
				assert.Equal(t, []byte(expectedField), field)
			}
			assert.False(t, it.HasNext())
		})
	}
}
