/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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
