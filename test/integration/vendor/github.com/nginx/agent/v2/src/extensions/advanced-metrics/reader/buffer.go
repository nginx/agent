package reader

import "io"

type fixedSizeBuffer struct {
	buffer []byte
	size   int
}

func NewFixedSizeBuffer(size int) *fixedSizeBuffer {
	return &fixedSizeBuffer{
		buffer: make([]byte, size),
	}
}

func (b *fixedSizeBuffer) get() []byte {
	return b.buffer[:b.size]
}

func (b *fixedSizeBuffer) readFrom(reader io.Reader) error {
	bytesReceived, err := reader.Read(b.buffer[b.size:])

	// https://pkg.go.dev/io#Reader
	// Always parse any received data and in case of EOF let the reader return EOF in next Read call.
	if bytesReceived <= 0 && err != nil {
		return err
	}
	b.size += bytesReceived
	return nil
}

func (b *fixedSizeBuffer) append(other []byte) int {
	copied := copy(b.buffer[b.size:], other)
	b.size += copied
	return copied
}

func (b *fixedSizeBuffer) clear() {
	b.size = 0
}
