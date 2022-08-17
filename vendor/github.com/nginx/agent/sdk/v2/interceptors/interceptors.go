package interceptors

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Interceptor interface {
	Stream() grpc.StreamClientInterceptor
	Unary() grpc.UnaryClientInterceptor
}

type ClientInterceptor interface {
	credentials.PerRPCCredentials
	Interceptor
}
