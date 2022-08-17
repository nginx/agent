package mocks

import (
	"net"
	"time"
)

type ConnStub struct {
	Repeats       int
	ReadBytes     int
	FramePosition int
	Separator     byte
}

func (s *ConnStub) Read(buff []byte) (int, error) {
	if s.Repeats == 0 {
		return 0, net.ErrClosed
	}
	s.Repeats--

	for i := range buff {
		buff[i] = 0
	}

	buff[s.FramePosition] = s.Separator

	return s.ReadBytes, nil
}

func (s *ConnStub) Close() error {
	return nil
}

func (s *ConnStub) LocalAddr() net.Addr {
	return nil
}

func (s *ConnStub) RemoteAddr() net.Addr {
	return nil
}

func (s *ConnStub) SetDeadline(time.Time) error {
	return nil
}

func (s *ConnStub) SetReadDeadline(time.Time) error {
	return nil
}

func (s *ConnStub) SetWriteDeadline(time.Time) error {
	return nil
}

func (s *ConnStub) Write([]byte) (int, error) {
	return 0, nil
}
