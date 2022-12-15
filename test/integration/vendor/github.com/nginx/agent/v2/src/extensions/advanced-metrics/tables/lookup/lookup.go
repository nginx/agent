/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package lookup

import (
	"errors"
	"sync"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
)

const (
	LookupNACode   = 0
	LookupAggrCode = 1

	lookupNA              = "NA"
	lookupAggr            = "AGGR"
	lookupAggrCode        = 1
	lookupFirstEntryIndex = 2
	maxLookupSize         = 1000000
	minLookupSize         = 4
)

type lookup struct {
	name string

	lock     sync.RWMutex
	data     []string
	hash     map[string]int
	dataSize int
}

func NewLookupFromSchema(fieldSchema *schema.Field) *lookup {
	return newLookup(fieldSchema.Name, fieldSchema.MaxDimensionSetSize)
}

func newLookup(name string, maxSize uint32) *lookup {
	if maxSize < minLookupSize {
		maxSize = minLookupSize
	}

	if maxSize > maxLookupSize {
		maxSize = maxLookupSize
	}

	l := &lookup{
		lock: sync.RWMutex{},
		name: name,

		data:     make([]string, maxSize),
		hash:     make(map[string]int, maxSize),
		dataSize: lookupFirstEntryIndex,
	}

	l.data[lookupAggrCode] = lookupAggr
	l.hash[lookupAggr] = lookupAggrCode

	l.data[LookupNACode] = lookupNA
	l.hash[lookupNA] = LookupNACode

	return l
}

// LookupBytes check if data is stored in lookup table and returns its code
// If data is not stored inside the lookup table it will be added to the table
func (l *lookup) LookupBytes(data []byte) int {
	l.lock.RLock()
	index, ok := l.hash[string(data)]
	l.lock.RUnlock()

	if ok {
		return index
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	index, ok = l.hash[string(data)]
	if ok {
		return index
	}

	index = lookupAggrCode
	if l.dataSize < len(l.data) {
		index = l.dataSize

		data := string(data)
		l.data[index] = data
		l.hash[data] = index
		l.dataSize++
	}
	return index
}

func (l *lookup) LookupCode(c int) (string, error) {
	if c > len(l.data) {
		return "", errors.New("code outside of lookup range")
	}

	l.lock.RLock()
	defer l.lock.RUnlock()

	if c >= l.dataSize {
		return "", errors.New("unknown code")
	}
	return l.data[c], nil
}

func (l *lookup) Name() string {
	return l.name
}
