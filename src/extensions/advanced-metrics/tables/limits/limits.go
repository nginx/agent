package limits

import (
	"fmt"
)

type CollapsingLevel = uint32

const (
	MaxCollapseLevel CollapsingLevel = 100
	NoCollapse       CollapsingLevel = MaxCollapseLevel + 10
)

type Limits struct {
	maxCapacity int
	threshold   int
}

func NewLimits(maxCapacity int, threshold int) (Limits, error) {
	if threshold > maxCapacity {
		return Limits{}, fmt.Errorf("threshold must be less than maxCapacity; threshold=%d maxCapacity=%d", threshold, maxCapacity)
	}
	if maxCapacity <= 0 {
		return Limits{}, fmt.Errorf("maxCapacity must be greater than 0; threshold=%d maxCapacity=%d", threshold, maxCapacity)
	}
	return Limits{
		maxCapacity: maxCapacity,
		threshold:   threshold,
	}, nil
}

func (l Limits) GetCurrentCollapsingLevel(currentSize int) CollapsingLevel {
	level := CollapsingLevel(0)
	if currentSize > l.threshold {
		aboveThresh := float32(currentSize - l.threshold)
		level = CollapsingLevel((aboveThresh / float32(l.maxCapacity-l.threshold)) * 100)
	}

	if level > MaxCollapseLevel {
		return MaxCollapseLevel
	}

	return level
}

func (l Limits) Threshold() int {
	return l.threshold
}

func (l Limits) Max() int {
	return l.maxCapacity
}
