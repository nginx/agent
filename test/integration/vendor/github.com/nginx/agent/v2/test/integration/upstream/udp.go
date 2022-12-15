package upstream

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"

	conf "github.com/nginx/agent/v2/test/integration/nginx"
	"github.com/stretchr/testify/assert"
)

type UdpHandler func(net.PacketConn, net.Addr, []byte)

const maxBufferSize = 1024

type UdpTestUpstream struct {
	Name    string
	Address string
	Handler UdpHandler
}

func (u *UdpTestUpstream) AsUpstream() conf.Upstream {
	return conf.Upstream{
		Server: u.Address,
	}
}

func (u *UdpTestUpstream) AsServer(address string, serverMarkers map[string]string, directives []string) conf.StreamServer {
	return conf.StreamServer{
		UpstreamName:     u.Name,
		Listen:           fmt.Sprintf("%s udp", address),
		F5MetricsMarkers: serverMarkers,
		Directives:       directives,
	}
}

func (u *UdpTestUpstream) Serve(t *testing.T) {
	mu := &sync.Mutex{}
	conn, err := net.ListenPacket("udp", u.Address)
	assert.NoError(t, err)

	go func() {
		for {
			mu.Lock()
			defer mu.Unlock()
			buff := make([]byte, maxBufferSize)
			size, addr, err := conn.ReadFrom(buff)
			if errors.Is(err, net.ErrClosed) {
				break
			}
			assert.NoError(t, err)
			go u.Handler(conn, addr, buff[:size])
		}
	}()

	t.Cleanup(func() {
		err := conn.Close()
		assert.NoError(t, err)
	})
}
