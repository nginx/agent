/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package service

import (
	"testing"

	"github.com/nginx/agent/v3/internal/model/os"
	"github.com/stretchr/testify/assert"
)

var processes = []*os.Process{
	{
		Pid:  123,
		Name: "nginx",
	},
}

func TestInstanceServiceUpdateProcesses(t *testing.T) {
	instanceService := NewInstanceService()
	instanceService.UpdateProcesses(processes)
	assert.Equal(t, processes, instanceService.processes)
}
