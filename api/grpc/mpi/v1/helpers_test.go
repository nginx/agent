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

func TestConvertToMaps(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]any
		input    []*structpb.Struct
	}{
		{
			name: "Test 1: Valid input with simple key-value pairs",
			expected: map[string]any{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			input: []*structpb.Struct{
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
		},
		{
			name: "Test 2: fractional number preserved as float64 whole number stays int",
			expected: map[string]any{
				"pi":    float64(3.14),
				"count": int(42),
			},
			input: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"pi": structpb.NewNumberValue(3.14),
					},
				},
				{
					Fields: map[string]*structpb.Value{
						"count": structpb.NewNumberValue(42),
					},
				},
			},
		},
		{
			name: "Test 3: Struct List and Null kinds are converted not dropped",
			expected: map[string]any{
				"nested":  map[string]any{"inner": "value"},
				"list":    []any{"a", float64(1)},
				"nil_val": nil,
			},
			input: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"nested": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"inner": structpb.NewStringValue("value"),
							},
						}),
					},
				},
				{
					Fields: map[string]*structpb.Value{
						"list": structpb.NewListValue(&structpb.ListValue{
							Values: []*structpb.Value{
								structpb.NewStringValue("a"),
								structpb.NewNumberValue(1),
							},
						}),
					},
				},
				{
					Fields: map[string]*structpb.Value{
						"nil_val": structpb.NewNullValue(),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertToMap(tt.input)

			assert.Equal(t, tt.expected, got)
		})
	}
}
