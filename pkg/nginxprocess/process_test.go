// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxprocess_test

import (
	"context"
	"os"
	"testing"

	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/require"
)

// gopsutilJunk is some junk that gopsutil CmdlineWithContext adds to the end of the command.
const gopsutilJunk = " ptr_munge= main_stack="

func TestList(t *testing.T) {
	t.Skipf("this test is only useful if you are running nginx locally")
	t.Parallel()
	p, err := nginxprocess.List(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, p)
}

func TestFind_PIDIsNotNginx(t *testing.T) {
	t.Parallel()
	p, err := nginxprocess.Find(context.Background(), int32(os.Getpid()))
	require.Error(t, err)
	require.Nil(t, p)
	require.True(t, nginxprocess.IsNotNginxErr(err))
}

func TestProcess_IsNginxWorker(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		cmd  string
		want bool
	}{
		"Test 1: nginx worker": {
			cmd:  "nginx: worker process",
			want: true,
		},
		"Test 2: nginx master": {
			cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			want: false,
		},
		"Test 3: nginx master with worker in command": {
			cmd:  "nginx: master process -c /foo/bar/workers/nginx.conf",
			want: false,
		},
		"Test 4: nginx cache manager": {
			cmd:  "nginx: cache manager process",
			want: false,
		},
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := nginxprocess.Process{Cmd: tc.cmd + gopsutilJunk}
			require.Equal(t, tc.want, p.IsWorker())
		})
	}
}

func TestProcess_IsNginxMaster(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		cmd  string
		want bool
	}{
		"Test 1: nginx worker": {
			cmd:  "nginx: worker process",
			want: false,
		},
		"Test 2: nginx master": {
			cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			want: true,
		},
		"Test 3: nginx master with worker in command": {
			cmd:  "nginx: master process -c /foo/bar/workers/nginx.conf",
			want: true,
		},
		"Test 4: nginx cache manager": {
			cmd:  "nginx: cache manager process",
			want: false,
		},
		"Test 5: nginx debug master": {
			cmd:  "{nginx-debug} nginx: master process /usr/sbin/nginx-debug -g daemon off;",
			want: true,
		},
		"Test 6: nginx debug worker": {
			cmd:  "{nginx-debug} nginx: worker process;",
			want: false,
		},
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := nginxprocess.Process{Cmd: tc.cmd + gopsutilJunk}
			require.Equal(t, tc.want, p.IsMaster())
		})
	}
}

func TestProcess_IsShuttingDown(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		cmd  string
		want bool
	}{
		"Test 1: nginx worker": {
			cmd:  "nginx: worker process",
			want: false,
		},
		"Test 2: nginx draining worker": {
			cmd:  "nginx: worker process is shutting down",
			want: true,
		},
		"Test 3: nginx master": {
			cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			want: false,
		},
	}
	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := nginxprocess.Process{Cmd: tc.cmd + gopsutilJunk}
			require.Equal(t, tc.want, p.IsShuttingDown())
		})
	}
}

func TestProcess_IsHealthy(t *testing.T) {
	t.Parallel()
	testcases := map[string]struct {
		status string
		want   bool
	}{
		"Test 1: Running": {
			status: process.Running,
			want:   true,
		},
		"Test 2: Blocked": {
			status: process.Blocked,
			want:   true,
		},
		"Test 3: Idle": {
			status: process.Idle,
			want:   true,
		},
		"Test 4: Lock": {
			status: process.Lock,
			want:   true,
		},
		"Test 5: Sleep": {
			status: process.Sleep,
			want:   true,
		},
		"Test 6: Stop": {
			status: process.Stop,
			want:   true,
		},
		"Test 7: Wait": {
			status: process.Wait,
			want:   true,
		},
		"Test 8: Daemon": {
			status: process.Daemon,
			want:   true,
		},
		"Test 9: Detached": {
			status: process.Detached,
			want:   true,
		},
		"Test 10: System": {
			status: process.System,
			want:   true,
		},
		"Test 11: Orphan": {
			status: process.Orphan,
			want:   true,
		},
		"Test 12: Zombie": {
			status: process.Zombie,
			want:   false,
		},
		"Test 13: empty string": {
			status: "",
			want:   false,
		},
		"Test 14: multiple healthy flags": {
			status: process.Running + " " + process.Lock,
			want:   true,
		},
		"Test 15: multiple flags ending in Zombie": {
			status: process.Running + " " + process.Zombie,
			want:   false,
		},
		"Test 16: multiple flags beginning in Zombie": {
			status: process.Zombie + " " + process.Running,
			want:   false,
		},
		"Test 17: multiple flags with Zombie in the middle": {
			status: process.Running + " " + process.Zombie + " " + process.Blocked,
			want:   false,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := nginxprocess.Process{Status: tc.status}
			require.Equal(t, tc.want, p.IsHealthy())
		})
	}
}

// BenchmarkList is useful if you are running NGINX locally and want to understand the performance impact of new changes
// to [nginxprocess.List].
func BenchmarkList(b *testing.B) {
	b.Skipf("skipping to prevent CI flake")
	ctx := context.Background()
	b.Run("base", func(b *testing.B) {
		for range b.N {
			_, err := nginxprocess.List(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("WithStatus", func(b *testing.B) {
		for range b.N {
			_, err := nginxprocess.List(ctx, nginxprocess.WithStatus(true))
			require.NoError(b, err)
		}
	})
}
