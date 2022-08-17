package reader

import (
	"bytes"
)

var (
	frameSeparatorByte = byte(';')
	frameSeparator     = []byte{frameSeparatorByte}
)

// frame is structure which contains messages read from the socket sent by the client.
//
// As frame is using preallocated buffer from shared pool in order to efficiently read data from the client
// it is important to call `Release` function after frame handling is done.
type frame struct {
	buffer    *fixedSizeBuffer
	frameSize int

	release func(*fixedSizeBuffer)
}

// Messages returns messages sent by the client.
//
// Single message contains only message data without messages separator.
func (f *frame) Messages() [][]byte {
	if f.buffer == nil {
		return nil
	}

	messages := bytes.Split(f.buffer.get()[:f.frameSize], frameSeparator)
	if len(messages) == 0 {
		return nil
	}
	trailingSlices := 1
	return messages[:len(messages)-trailingSlices]
}

// Release clears Frame internal data and returns buffer to the pool.
// This function should be always called after Frame handling is done.
// This is safe to call any other methods of Frame after Release call.
func (f *frame) Release() {
	f.buffer.clear()
	f.release(f.buffer)
	f.frameSize = 0
	f.buffer = nil
}
