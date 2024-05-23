// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package watcher

import (
	"context"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

func TestNginxHealthWatcherOperator_Health(t *testing.T) {
	ctx := context.Background()
	nginxHealthWatcher := NewNginxHealthWatcher()
	instanceID := protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()

	expected := &v1.InstanceHealth{
		InstanceId:           instanceID,
		Description:          "instance is healthy",
		InstanceHealthStatus: 1,
	}

	instanceHealth := nginxHealthWatcher.Health(ctx, instanceID)

	assert.Equal(t, expected, instanceHealth)
}
