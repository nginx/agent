// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"sort"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/stretchr/testify/assert"
)

func CompareInstances(t testing.TB, expected, result map[string]*v1.Instance) {
	t.Helper()
	for _, instance := range result {
		instanceID := instance.GetInstanceMeta().GetInstanceId()
		if instance.GetInstanceRuntime().GetNginxRuntimeInfo() != nil {
			sort.Strings(instance.GetInstanceRuntime().GetNginxRuntimeInfo().GetDynamicModules())
			assert.Equal(t, expected[instanceID].GetInstanceRuntime().GetNginxRuntimeInfo(),
				instance.GetInstanceRuntime().GetNginxRuntimeInfo())
		} else {
			sort.Strings(instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetDynamicModules())
			assert.Equal(t, expected[instanceID].GetInstanceRuntime().GetNginxPlusRuntimeInfo(),
				instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo())
		}
	}
}
