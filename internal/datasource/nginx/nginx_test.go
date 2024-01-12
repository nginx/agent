/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nginx

import (
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/datasource/nginx/process"
	"github.com/nginx/agent/v3/internal/model/os"
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

	tests := []struct {
		name     string
		input    *process.Info
		expected []*instances.Instance
	}{
		{
			name: "NGINX open source",
			input: &process.Info{
				Version:     "1.23.3",
				PlusVersion: "",
				Prefix:      "/usr/local/Cellar/nginx/1.23.3",
				ConfPath:    "/usr/local/etc/nginx/nginx.conf",
			},
			expected: []*instances.Instance{
				{
					InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e",
					Type:       instances.Type_NGINX,
					Version:    "1.23.3",
				},
			},
		}, {
			name: "NGINX plus",
			input: &process.Info{
				Version:     "",
				PlusVersion: "R30",
				Prefix:      "/usr/local/Cellar/nginx/1.23.3",
				ConfPath:    "/usr/local/etc/nginx/nginx.conf",
			},
			expected: []*instances.Instance{
				{
					InstanceId: "ae6c58c1-bc92-30c1-a9c9-85591422068e",
					Type:       instances.Type_NGINXPLUS,
					Version:    "R30",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			n := New(NginxParameters{
				GetInfo: func(pid int32, exe string) (*process.Info, error) {
					return test.input, nil
				},
			})
			result, err := n.GetInstances(processes)

			assert.Equal(t, test.expected, result)
			assert.NoError(t, err)
		})
	}
}
