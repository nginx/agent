// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import "context"

const (
	TokenKey = "token"
)

// PerRPCCredentials implements the PerRPCCredentials interface.
type PerRPCCredentials struct {
	Token string
}

// GetRequestMetadata returns the request metadata as a map.
func (t *PerRPCCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		TokenKey: t.Token,
	}, nil
}

// RequireTransportSecurity indicates whether the credentials requires transport security.
func (t *PerRPCCredentials) RequireTransportSecurity() bool {
	return true
}