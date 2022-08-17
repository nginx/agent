package priority_table

import (
	"container/heap"
	"testing"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/stretchr/testify/assert"
)

func TestSampleQueue(t *testing.T) {
	queue := &sampleQueue{}

	sample1 := newTestSample(1)
	sample2 := newTestSample(2)
	sample3 := newTestSample(3)
	heap.Push(queue, sample2)
	heap.Push(queue, sample1)
	heap.Push(queue, sample3)
	assert.Equal(t, sample1, queue.Peek())
	assert.Equal(t, sample1, heap.Pop(queue))

	assert.Equal(t, sample2, queue.Peek())
	assert.Equal(t, sample2, heap.Pop(queue))

	assert.Equal(t, sample3, queue.Peek())
	assert.Equal(t, sample3, heap.Pop(queue))
}

func newTestSample(hitcount int) *sample.Sample {
	s := sample.NewSample(0, 0)
	s.AddHitCount(hitcount)
	return &s
}
