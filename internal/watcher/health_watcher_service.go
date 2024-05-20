// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// nolint
type (
	healthWatcherOperator interface {
		Watch(ctx context.Context) Health
	}

	Health struct {
		healthStatus *v1.InstanceHealth_InstanceHealthStatus
		description  string
	}

	HealthWatcherService struct {
		cache    map[string]Health // key is instanceID
		watchers map[string]healthWatcherOperator
	}
)

func NewHealthWatcherService() *HealthWatcherService {
	return &HealthWatcherService{
		cache:    make(map[string]Health),
		watchers: make(map[string]healthWatcherOperator),
	}
}

func (hw *HealthWatcherService) AddHealthWatcher(instanceID string) {
}
