package lookup

import (
	"fmt"
	"sync"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
)

type Lookuper interface {
	Name() string
	LookupBytes([]byte) int
	LookupCode(int) (string, error)
}

type LookupSet struct {
	lock    sync.RWMutex
	lookups []Lookuper
}

func NewLookupSetFromSchema(dimensions []*schema.Field) *LookupSet {
	lookups := make([]Lookuper, 0, len(dimensions))
	for _, dimension := range dimensions {
		lookups = append(lookups, NewLookupFromSchema(dimension))
	}
	return newLookupSet(lookups...)
}

func newLookupSet(lookups ...Lookuper) *LookupSet {
	return &LookupSet{
		lock:    sync.RWMutex{},
		lookups: lookups,
	}
}

func (s *LookupSet) Len() int {
	return len(s.lookups)
}

func (s *LookupSet) Name(lookupIndex schema.FieldIndex) (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if err := s.validateLookupIndex(lookupIndex); err != nil {
		return "", err
	}
	return s.lookups[lookupIndex].Name(), nil
}

func (s *LookupSet) LookupBytes(lookupIndex schema.FieldIndex, k []byte) (int, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if err := s.validateLookupIndex(lookupIndex); err != nil {
		return 0, err
	}
	return s.lookups[lookupIndex].LookupBytes(k), nil
}

func (s *LookupSet) LookupCode(lookupIndex schema.FieldIndex, k int) (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if err := s.validateLookupIndex(lookupIndex); err != nil {
		return "", err
	}
	return s.lookups[lookupIndex].LookupCode(k)
}

func (s *LookupSet) validateLookupIndex(lookupIndex schema.FieldIndex) error {
	if int(lookupIndex) >= len(s.lookups) {
		return fmt.Errorf("lookup key '%d' out of maximum range: %d", lookupIndex, len(s.lookups))
	}
	return nil
}
