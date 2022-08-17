package sample

import "fmt"

// SampleTableKey is composite key which is constructed from multiple integers.
// Each key part integer is encoded into byte array with bit size accuracy.
type SampleKey struct {
	key      []byte
	bitsUsed int
}

func NewSampleKey(size int) SampleKey {
	return SampleKey{
		key:      make([]byte, size),
		bitsUsed: 0,
	}
}

func (s *SampleKey) SetKeyPart(keyPart int, keyBitSize int, keyStartPosition int) {
	bitsUsed := keyStartPosition + keyBitSize
	keyPart &= (1 << keyBitSize) - 1

	byteIndex := (bitsUsed) / 8
	firstByteUsedBits := bitsUsed % 8
	keyFirstByteShift := 8 - firstByteUsedBits
	keyFirstByteVal := (keyPart & mask(firstByteUsedBits)) << keyFirstByteShift

	maskSize := firstByteUsedBits
	if keyBitSize < firstByteUsedBits {
		maskSize = keyBitSize
	}

	keyPartZeroingMask := byte(^(mask(maskSize) << keyFirstByteShift))
	s.key[byteIndex] &= keyPartZeroingMask
	s.key[byteIndex] |= byte(keyFirstByteVal)
	keyBitSize -= firstByteUsedBits
	byteIndex--
	keyPart >>= firstByteUsedBits

	for keyBitSize > 8 {
		s.key[byteIndex] = byte(keyPart & mask(8))

		keyBitSize -= 8
		keyPart >>= 8
		byteIndex--
	}

	if keyBitSize > 0 {
		keyPartZeroingMask := byte(^mask(keyBitSize))
		s.key[byteIndex] &= keyPartZeroingMask
		s.key[byteIndex] |= byte(keyPart)
	}
}

func (s *SampleKey) AddKeyPart(keyPart int, keyBitSize int) error {
	if s.bitsUsed+keyBitSize > len(s.key)*8 {
		return fmt.Errorf("key part bit size %d exceeded table key size %d", keyBitSize, len(s.key))
	}

	s.SetKeyPart(keyPart, keyBitSize, s.bitsUsed)
	s.bitsUsed += keyBitSize

	return nil
}

func (s *SampleKey) GetKeyParts(partSizes []int) []int {
	keyParts := make([]int, 0, len(partSizes))

	bitOffset := 0
	for _, size := range partSizes {
		byteIndex := bitOffset / 8
		lastByteNumberOfBits := 8 - bitOffset%8

		bitOffset += size

		if size < lastByteNumberOfBits {
			singleByteKeyShift := lastByteNumberOfBits - size
			singleByteKeyPart := (int(s.key[byteIndex]) >> singleByteKeyShift) & mask(size)
			keyParts = append(keyParts, singleByteKeyPart)
			continue
		}

		keyPart := int(s.key[byteIndex]) & mask(lastByteNumberOfBits)
		byteIndex++
		size -= lastByteNumberOfBits

		for size > 8 {
			keyPart <<= 8
			keyPart |= int(s.key[byteIndex])
			byteIndex++
			size -= 8
		}

		if size > 0 {
			keyPart <<= size
			keyPart |= int(s.key[byteIndex] >> (8 - size))
		}

		keyParts = append(keyParts, keyPart)
	}

	return keyParts
}

func (s *SampleKey) AsByteKey() []byte {
	return s.key
}

func (s *SampleKey) AsStringKey() string {
	return string(s.key)
}

func mask(size int) int {
	return (1 << size) - 1
}
