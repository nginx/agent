package sample

import (
	cmap "github.com/orcaman/concurrent-map"
)

type sampleTable struct {
	cdata cmap.ConcurrentMap
}

func NewSampleTable() *sampleTable {
	return &sampleTable{
		cdata: cmap.New(),
	}
}

func (st *sampleTable) Len() int {
	return st.cdata.Count()
}

func (st *sampleTable) Add(sampleToAdd Sample) error {
	var err error
	st.cdata.Upsert(sampleToAdd.Key().AsStringKey(), &sampleToAdd, func(exist bool, valueInMap interface{}, newValue interface{}) interface{} {
		if !exist {
			return newValue
		}
		sampleInMap := valueInMap.(*Sample)
		err = sampleInMap.AddSample(&sampleToAdd)
		return sampleInMap
	})
	return err
}

func (st *sampleTable) Range(cb func(s *Sample)) {
	st.cdata.IterCb(func(k string, v interface{}) {
		val := v.(*Sample)
		cb(val)
	})
}

func (st *sampleTable) Clear() {
	st.cdata.Clear()
}
