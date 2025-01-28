// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestConvertToStructs(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []*structpb.Struct
		wantErr  bool
	}{
		{
			name: "Test 1: Valid input with simple key-value pairs",
			input: map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			expected: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"key1": structpb.NewStringValue("value1"),
					},
				},
				{
					Fields: map[string]*structpb.Value{
						"key2": structpb.NewNumberValue(123),
					},
				},
				{
					Fields: map[string]*structpb.Value{
						"key3": structpb.NewBoolValue(true),
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "Test 2: Empty input map",
			input:    make(map[string]any),
			expected: []*structpb.Struct{},
			wantErr:  false,
		},
		{
			name: "Test 3: Invalid input type",
			input: map[string]any{
				"key1": func() {}, // Unsupported type
			},
			expected: []*structpb.Struct{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToStructs(tt.input)

			assert.ElementsMatch(t, tt.expected, got)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
