/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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
