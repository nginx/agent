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
			Pid:  2,
			Ppid: 1,
			Name: "test",
			Cmd:  "test -start",
			Exe:  "/bin/test",
		}, {
			Pid:  3,
			Ppid: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			Pid:  4,
			Ppid: 3,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			Pid:  5,
			Ppid: 3,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			Pid:  6,
			Ppid: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			Pid:  7,
			Ppid: 6,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		}, {
			Pid:  8,
			Ppid: 6,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
		},
	}

	for i := 0; i < b.N; i++ {
		getNginxProcesses(newProcesses)
	}
}
