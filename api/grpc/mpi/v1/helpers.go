// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package v1

import (
	"log/slog"

	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertToStructs converts a map[string]any into a slice of *structpb.Struct.
// Each key-value pair in the input map is converted into a *structpb.Struct,
// where the key is used as the field name, and the value is added to the Struct.
//
// Parameters:
//   - input: A map[string]any containing key-value pairs to be converted.
//
// Returns:
//   - []*structpb.Struct: A slice of *structpb.Struct, where each map entry is converted into a struct.
//   - error: An error if any value in the input map cannot be converted into a *structpb.Struct.
//
// Example:
//
//	input := map[string]any{
//	    "key1": "value1",
//	    "key2": 123,
//	    "key3": true,
//	}
//	structs, err := ConvertToStructs(input)
//	// structs will contain a slice of *structpb.Struct
//	// err will be nil if all conversions succeed.
func ConvertToStructs(input map[string]any) ([]*structpb.Struct, error) {
	structs := []*structpb.Struct{}
	for key, value := range input {
		// Convert each value in the map to *structpb.Struct
		structValue, err := structpb.NewStruct(map[string]any{
			key: value,
		})
		if err != nil {
			return structs, err
		}
		structs = append(structs, structValue)
	}

	return structs, nil
}

func ConvertToMap(input []*structpb.Struct) map[string]any {
	convertedMap := make(map[string]any)
	for _, value := range input {
		for key, field := range value.GetFields() {
			kind := field.GetKind()
			switch kind.(type) {
			case *structpb.Value_StringValue:
				convertedMap[key] = field.GetStringValue()
			case *structpb.Value_NumberValue:
				convertedMap[key] = int(field.GetNumberValue())
			case *structpb.Value_BoolValue:
				convertedMap[key] = field.GetBoolValue()
			default:
				slog.Warn("Unknown type for map conversion", "value", kind)
			}
		}
	}

	return convertedMap
}
