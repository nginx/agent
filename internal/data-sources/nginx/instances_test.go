package nginx_test

import (
	"testing"

	"github.com/nginx/agent/v3/internal/data-sources/nginx"
	"github.com/nginx/agent/v3/internal/models/instances"
	"github.com/nginx/agent/v3/internal/models/os"
	"github.com/stretchr/testify/assert"
)

func TestGetInstances(t *testing.T) {
	processes := []*os.Process{
		{
			Pid:  123,
			Ppid: 456,
			Name: "nginx",
			Cmd:  "nginx: worker process",
		},
		{
			Pid:  789,
			Ppid: 123,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
		},
		{
			Pid:  543,
			Ppid: 454,
			Name: "grep",
			Cmd:  "grep --color=auto --exclude-dir=.bzr --exclude-dir=CVS --exclude-dir=.git --exclude-dir=.hg --exclude-dir=.svn --exclude-dir=.idea --exclude-dir=.tox nginx",
		},
	}
	result, err := nginx.GetInstances(processes)
	expected := []*instances.Instance{
		{
			InstanceId: "789",
			Type:       instances.Type_NGINX,
		},
	}

	assert.Equal(t, expected, result)
	assert.NoError(t, err)
}
