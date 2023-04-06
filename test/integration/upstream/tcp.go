package upstream

import (
	"errors"
	"net"
	"sync"
	"testing"

	conf "github.com/nginx/agent/test/integration/nginx"
	"github.com/stretchr/testify/assert"
)

type TcpHandler func(conn net.Conn)

type TcpTestUpstream struct {
	Name    string
	Address string
	Handler TcpHandler
}

func (u *TcpTestUpstream) AsUpstream() conf.Upstream {
	return conf.Upstream{
		Server: u.Address,
	}
}

func (u *TcpTestUpstream) AsServer(address string, serverMarkers map[string]string) conf.StreamServer {
	return conf.StreamServer{
		UpstreamName:     u.Name,
		Listen:           address,
		F5MetricsMarkers: serverMarkers,
	}
}

func (u *TcpTestUpstream) Serve(t *testing.T) {
	mu := &sync.Mutex{}
	l, err := net.Listen("tcp", u.Address)
	assert.NoError(t, err)

	go func() {
		for {
			mu.Lock()
			defer mu.Unlock()
			conn, err := l.Accept()
			if errors.Is(err, net.ErrClosed) {
				break
			}
			assert.NoError(t, err)
			go u.Handler(conn)
		}
	}()

	t.Cleanup(func() {
		err := l.Close()
		assert.NoError(t, err)
	})
}
