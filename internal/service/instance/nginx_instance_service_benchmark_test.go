// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bytes"
	"fmt"
	"github.com/nginx/agent/v3/internal/model"
	"testing"
)

func BenchmarkNginxService_getNginxProcesses(b *testing.B) {
	nginxService := NewNginx(NginxParameters{})

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
		nginxService.getNginxProcesses(newProcesses)
	}
}

func BenchmarkNginxService_parseNginxVersionCommandOutput(b *testing.B) {
	output := fmt.Sprintf(`nginx version: nginx/1.23.3
	built by clang 14.0.0 (clang-1400.0.29.202)
	built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
	TLS SNI support enabled
	configure arguments: %s`, ossConfigArgs)

	for i := 0; i < b.N; i++ {
		parseNginxVersionCommandOutput(bytes.NewBufferString(output))
	}
}
