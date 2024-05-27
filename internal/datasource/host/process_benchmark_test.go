// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"testing"

	"github.com/nginx/agent/v3/internal/model"
)

func BenchmarkNginxService_getNginxProcesses(b *testing.B) {
	newProcesses := []*model.Process{
		{
			PID:  2,
			PPID: 1,
			Name: "test",
			Cmd:  "test -start",
			Exe:  "/bin/test",
		}, {
			PID:  3,
			PPID: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			PID:  4,
			PPID: 3,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			PID:  5,
			PPID: 3,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			PID:  6,
			PPID: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			PID:  7,
			PPID: 6,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			PID:  8,
			PPID: 6,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		},
	}

	for i := 0; i < b.N; i++ {
		findNginxProcesses(newProcesses)
	}
}
