package client

import (
	"google.golang.org/grpc"

	"github.com/nginx/agent/sdk/v2/interceptors"
)

type connector struct {
	server             string            //nolint:structcheck,unused
	dialOptions        []grpc.DialOption //nolint:structcheck,unused
	interceptors       []interceptors.Interceptor
	clientInterceptors []interceptors.ClientInterceptor
	grpc               *grpc.ClientConn
}

func newConnector() *connector {
	return &connector{}
}
