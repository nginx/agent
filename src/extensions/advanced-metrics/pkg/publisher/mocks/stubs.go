/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package mocks

import (
	"errors"
	"fmt"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
)

type LookupSetStub struct {
	Lookups map[int]map[int]string
}

func (LookupSetStub) LookupBytes(schema.FieldIndex, []byte) (int, error) {
	return 0, errors.New("LookupSetStub::LookupBytes not implemented")
}

func (l *LookupSetStub) LookupCode(index int, code int) (string, error) {
	values, ok := l.Lookups[index]
	if !ok {
		return "", fmt.Errorf("index %d not found", index)
	}

	value, ok := values[code]
	if !ok {
		return "", fmt.Errorf("code %d not found", code)
	}
	return value, nil
}

type PriorityTableStub struct {
	SamplesMap map[string]*sample.Sample
}

func (p *PriorityTableStub) Samples() map[string]*sample.Sample {
	return p.SamplesMap
}
