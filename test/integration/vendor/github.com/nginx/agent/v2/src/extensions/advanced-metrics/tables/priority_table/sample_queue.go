package priority_table

import (
	"container/heap"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
)

type sampleQueue []*sample.Sample

var _ heap.Interface = &sampleQueue{}

func (q sampleQueue) Len() int {
	return len(q)
}

func (q sampleQueue) Less(i, j int) bool {
	return q[i].HitCount() < q[j].HitCount()
}

func (q sampleQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q sampleQueue) Peek() *sample.Sample {
	return q[0]
}

func (q sampleQueue) ReplaceTop(s *sample.Sample) {
	q[0] = s
}

func (q *sampleQueue) Pop() interface{} {
	o := *q
	n := len(o)
	x := o[n-1]
	*q = o[0 : n-1]
	return x
}

func (q *sampleQueue) Push(s interface{}) {
	*q = append(*q, s.(*sample.Sample))
}
