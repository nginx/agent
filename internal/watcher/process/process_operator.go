// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"

	"github.com/nginx/agent/v3/pkg/nginxprocess"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ProcessOperatorInterface
type (
	// ProcessOperator provides details about running NGINX processes.
	ProcessOperator struct{}

	ProcessOperatorInterface interface {
		Processes(ctx context.Context) ([]*nginxprocess.Process, error)
		Process(ctx context.Context, pid int32) (*nginxprocess.Process, error)
	}
)

var _ ProcessOperatorInterface = (*ProcessOperator)(nil)

func NewProcessOperator() *ProcessOperator {
	return &ProcessOperator{}
}

func (pw *ProcessOperator) Processes(ctx context.Context) ([]*nginxprocess.Process, error) {
	return nginxprocess.List(ctx)
}

func (pw *ProcessOperator) Process(ctx context.Context, pid int32) (*nginxprocess.Process, error) {
	return nginxprocess.Find(ctx, pid, nginxprocess.WithStatus(true))
}
