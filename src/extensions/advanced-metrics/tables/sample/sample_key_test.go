/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sample

import (
	"crypto/rand"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSampleKeyAddKeyPart(t *testing.T) {
	type keyDesc struct {
		bits  int
		value int
	}
	tcs := []struct {
		name   string
		values []keyDesc
	}{
		{
			name:   "single key",
			values: []keyDesc{{2, 1}},
		},
		{
			name:   "single key, not full 2 bytes",
			values: []keyDesc{{15, 0x70a0}},
		},
		{
			name:   "single key, full 2 bytes",
			values: []keyDesc{{16, 0xf0a0}},
		},
		{
			name:   "single key, not full multibytes",
			values: []keyDesc{{25, 0x1f0a0b1}},
		},
		{
			name:   "single key, full multibytes",
			values: []keyDesc{{24, 0xf0a0b1}},
		},
		{
			name:   "multi key, no full bytes",
			values: []keyDesc{{4, 0xa}, {4, 0xb}},
		},
		{
			name:   "multi key, 2 byte second key",
			values: []keyDesc{{5, 0xa}, {8, 0xab}},
		},
		{
			name:   "multi key, multi byte second key",
			values: []keyDesc{{5, 0xa}, {16, 0xabba}},
		},
		{
			name:   "multi key, 5 byte second key",
			values: []keyDesc{{5, 0xa}, {32, 0xabbaacca}},
		},
		{
			name:   "multi key, 3 keys in byte",
			values: []keyDesc{{4, 0xa}, {2, 0x2}, {5, 0xc}},
		},
		{
			name:   "multi key, all keys share bytes",
			values: []keyDesc{{4, 0xa}, {8, 0x2}, {9, 0xc}, {9, 0xc}},
		},
		{
			name:   "multi key, 4 keys in byte",
			values: []keyDesc{{1, 0x1}, {1, 0x0}, {3, 0x5}, {1, 0x1}},
		},
	}

	const maxKeySize = 125
	for _, tc := range tcs {
		key := NewSampleKey(maxKeySize)

		keysSizes := make([]int, 0)
		keysValues := make([]int, 0)

		for _, kd := range tc.values {
			err := key.AddKeyPart(kd.value, kd.bits)
			assert.NoError(t, err)

			keysSizes = append(keysSizes, kd.bits)
			keysValues = append(keysValues, kd.value)
		}

		assert.Equal(t, keysValues, key.GetKeyParts(keysSizes))
	}
}

func TestSampleKeyGetKeyPartsWithGenerator(t *testing.T) {
	const keyPartMaxBitSize = 31
	const maxSampleTableKeySize = 3000

	for i := 0; i < 2000; i++ {
		var sampleTableKeySize int = 0
		s := NewSampleKey(maxSampleTableKeySize)

		keysSizes := make([]int, 0)
		keysValues := make([]int, 0)

		for sampleTableKeySize < maxSampleTableKeySize-keyPartMaxBitSize*3 {
			size, _ := rand.Int(rand.Reader, big.NewInt(keyPartMaxBitSize))
			size.Add(size, big.NewInt(1))

			maxValue := int64(math.Pow(2, float64(size.Int64()))) - 1
			val, _ := rand.Int(rand.Reader, big.NewInt(maxValue))

			err := s.AddKeyPart(int(val.Int64()), int(size.Int64()))
			assert.NoError(t, err)
			sampleTableKeySize += int(size.Int64())

			keysSizes = append(keysSizes, int(size.Int64()))
			keysValues = append(keysValues, int(val.Int64()))

		}
		assert.Equal(t, keysValues, s.GetKeyParts(keysSizes))
	}
}

func TestSampleKeySetKeyPart(t *testing.T) {
	keySize := 100

	key := NewSampleKey(keySize)
	err := key.AddKeyPart(11, 10)
	assert.NoError(t, err)
	err = key.AddKeyPart(12, 10)
	assert.NoError(t, err)
	err = key.AddKeyPart(13, 10)
	assert.NoError(t, err)

	assert.Equal(t, []int{11, 12, 13}, key.GetKeyParts([]int{10, 10, 10}))

	key.SetKeyPart(0xf1, 10, 0)
	key.SetKeyPart(0xf2, 10, 10)
	key.SetKeyPart(0xf4, 10, 20)

	assert.Equal(t, []int{0xf1, 0xf2, 0xf4}, key.GetKeyParts([]int{10, 10, 10}))
}

func TestSampleKeySetKeyPartGenerated(t *testing.T) {
	// This makes test predictable and generally speaking this is a generator of numbers
	const keyPartMaxBitSize = 31
	const maxSampleTableKeySize = 3000

	for i := 0; i < 2000; i++ {
		var sampleTableKeySize int = 0
		s := NewSampleKey(maxSampleTableKeySize)

		keysSizes := make([]int, 0)

		for sampleTableKeySize < maxSampleTableKeySize-keyPartMaxBitSize*3 {
			size, _ := rand.Int(rand.Reader, big.NewInt(keyPartMaxBitSize))
			size.Add(size, big.NewInt(1))

			maxValue := int64(math.Pow(2, float64(size.Int64()))) - 1
			val, _ := rand.Int(rand.Reader, big.NewInt(maxValue))
			err := s.AddKeyPart(int(val.Int64()), int(size.Int64()))
			assert.NoError(t, err)
			sampleTableKeySize += int(size.Int64())

			keysSizes = append(keysSizes, int(size.Int64()))

		}

		keysValues := make([]int, 0)
		startPosition := 0
		for _, size := range keysSizes {
			maxValue := int64(math.Pow(2, float64(size))) - 1
			val, _ := rand.Int(rand.Reader, big.NewInt(maxValue))

			s.SetKeyPart(int(val.Int64()), size, startPosition)
			sampleTableKeySize += size

			keysValues = append(keysValues, int(val.Int64()))
			startPosition += size

		}
		assert.Equal(t, keysValues, s.GetKeyParts(keysSizes))
	}
}
