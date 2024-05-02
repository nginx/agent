package host

import (
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcess_GetNginxProcesses(t *testing.T) {
	tests := []struct {
		name      string
		processes []*model.Process
		expected  NginxProcesses
	}{
		{
			name: "Test 1: One NGINX Process ",
			processes: []*model.Process{
				{
					Pid:  2,
					Ppid: 1,
					Name: "test",
					Cmd:  "test -start",
					Exe:  "/bin/test",
				},
				{
					Pid:  3,
					Ppid: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
				},
			},
			expected: NginxProcesses{
				3: &model.Process{
					Pid:  3,
					Ppid: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
				},
			},
		},
		{
			name: "Test 2: No NGINX Process ",
			processes: []*model.Process{
				{
					Pid:  2,
					Ppid: 1,
					Name: "test",
					Cmd:  "test -start",
					Exe:  "/bin/test",
				},
			},
			expected: NginxProcesses{},
		},
		{
			name: "Test 3: Upgrade NGINX Process ",
			processes: []*model.Process{
				{
					Pid:  2,
					Ppid: 1,
					Name: "nginx",
					Cmd:  "nginx: upgrade",
					Exe:  "/usr/local/Cellar/nginx/1.23.3/bin/nginx",
				},
			},
			expected: NginxProcesses{},
		},

		{
			name: "Test 4: Non NGINX Process ",
			processes: []*model.Process{
				{
					Pid:  2,
					Ppid: 1,
					Name: "nginx",
					Cmd:  "/usr/sbin/nginx-asg-sync -log_path=/var/log/nginx-asg-sync/nginx-asg-sync.log",
					Exe:  "",
				},
			},
			expected: NginxProcesses{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			results := findNginxProcesses(test.processes)
			assert.Equal(tt, test.expected, results)
		})
	}
}
