/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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
