package mocks

type FieldIteratorStub struct {
	data    [][]byte
	current int
}

func NewFieldIteratorStub(data [][]byte) *FieldIteratorStub {
	return &FieldIteratorStub{
		data:    data,
		current: 0,
	}
}

func (f *FieldIteratorStub) Next() []byte {
	if !f.HasNext() {
		return nil
	}
	res := f.data[f.current]
	f.current++
	return res
}
func (f *FieldIteratorStub) HasNext() bool {
	return f.current < len(f.data)
}
