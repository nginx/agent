// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"sort"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

func SortInstanceChildren(children []*v1.InstanceChild) []*v1.InstanceChild {
	sort.Slice(children, func(i, j int) bool {
		return children[i].GetProcessId() > children[j].GetProcessId()
	})

	return children
}
