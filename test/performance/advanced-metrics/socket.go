package main

import (
	"net"
	"time"
)

type Pusher struct {
	socket string
	conn   net.Conn
}

func NewPusher(addr string) *Pusher {
	return &Pusher{socket: addr}
}

func (p *Pusher) Connect() error {
	con, err := net.Dial("unix", p.socket)
	if err != nil {
		return err
	}
	p.conn = con
	return nil
}

func (p *Pusher) Close() {
	_ = p.conn.Close()
}

func (p *Pusher) PushToSocket(message []byte, startFrom int, timeout time.Duration) (int, error) {
	err := p.conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return 0, err
	}
	sent, err := p.conn.Write(message[startFrom:])
	if err != nil {
		return sent, err
	}
	return sent, nil
}
