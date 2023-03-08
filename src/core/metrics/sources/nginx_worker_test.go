/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	host          = "node_id"
	displayName   = "node_displayname"
	instanceGroup = "my_instances"
	nginxId       = uuid.New().String()
	systemId      = uuid.New().String()
	procMap       = map[int32][]int32{
		2: {
			1,
		},
	}
)

type MockWorkerClient struct {
	mock.Mock
}

func (m *MockWorkerClient) GetWorkerStats(childProcs []int32) (*WorkerStats, error) {
	args := m.Called(childProcs)
	return args.Get(0).(*WorkerStats), nil
}

func NewMockWorkerClient() *MockWorkerClient {
	return &MockWorkerClient{}
}

func TestNginxWorkerCollector(t *testing.T) {
	dimensions := &metrics.CommonDim{
		SystemId:      systemId,
		Hostname:      host,
		InstanceGroup: instanceGroup,
		DisplayName:   displayName,
		NginxId:       nginxId,
	}

	mockBinary := &utils.MockNginxBinary{}
	mockClient := &MockWorkerClient{}

	n := NewNginxWorker(dimensions, OSSNamespace, mockBinary, mockClient)

	mockBinary.On("GetChildProcesses").Return(procMap)

	mockClient.On("GetWorkerStats", procMap[2]).Return(&WorkerStats{
		Workers: &Workers{
			Count:        1.00,
			KbsR:         1.00,
			KbsW:         1.00,
			CPUSystem:    1.00,
			CPUTotal:     1.00,
			CPUUser:      1.00,
			FdsCount:     1.00,
			MemVms:       1.00,
			MemRss:       1.00,
			MemRssPct:    1.00,
			RlimitNofile: 1.00,
		},
	}, nil)

	// tell the mock nginx binary to return something
	ctx := context.TODO()

	wg := sync.WaitGroup{}
	wg.Add(1)
	m := make(chan *proto.StatsEntity)
	go n.Collect(ctx, &wg, m)

	time.Sleep(100 * time.Millisecond)
	mockClient.AssertNumberOfCalls(t, "GetWorkerStats", 2)
	mockBinary.AssertNumberOfCalls(t, "GetChildProcesses", 1)

	// prev stats will initially all be 0 because sync.Once will set
	// the prev stats as equal to the initial stats collected
	// that's ok, but we should test the counter gauge computations again later
	metricReport := <-m
	for _, metric := range metricReport.Simplemetrics {
		switch metric.Name {
		case "nginx.workers.cpu.system":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.cpu.total":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.io.kbs_w":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.io.kbs_r":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.mem.rss":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.mem.rss_pct":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.rlimit_nofile":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.count":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.mem.vms":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.fds_count":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.cpu.user":
			assert.Equal(t, float64(0), metric.Value)
		default:
			// if there is an unknown metric, we should fail because
			// we should't have anything but the above
			assert.Fail(t, "saw an unknown metric in test")
		}
	}

	wg.Add(1)

	go n.Collect(ctx, &wg, m)

	time.Sleep(100 * time.Millisecond)
	mockClient.AssertNumberOfCalls(t, "GetWorkerStats", 3)
	mockBinary.AssertNumberOfCalls(t, "GetChildProcesses", 2)

	metricReport = <-m
	for _, metric := range metricReport.Simplemetrics {
		switch metric.Name {
		case "nginx.workers.cpu.system":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.cpu.total":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.io.kbs_w":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.io.kbs_r":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.mem.rss":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.mem.rss_pct":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.rlimit_nofile":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.count":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.mem.vms":
			assert.Equal(t, float64(0), metric.Value)
		case "nginx.workers.fds_count":
			assert.Equal(t, float64(1), metric.Value)
		case "nginx.workers.cpu.user":
			assert.Equal(t, float64(0), metric.Value)
		default:
			// if there is an unknown metric, we should fail because
			// we should't have anything but the above
			assert.Fail(t, "saw an unknown metric in test")
		}
	}
}

func TestExcludeCacheProcs(t *testing.T) {
	type testCase struct {
		input    []byte
		expected map[string]bool
	}

	tests := map[string]testCase{
		"sample ps input should exclude the cache manager proc 3341323": {
			input: []byte(`naas     1571264  0.1  2.0  97604 80636 ?        S    00:00   1:02 [celeryd: w0@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w0.log --pidfile=/var/run/naas/queue-w0.pid --hostname=w0@test.server.com)
			naas     1571284  0.1  2.0  97676 80952 ?        S    00:00   1:01 [celeryd: w1@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w1.log --pidfile=/var/run/naas/queue-w1.pid --hostname=w1@test.server.com)
			naas     1571325  0.0  1.8  96788 72148 ?        S    00:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-1]
			naas     1571342  0.0  1.8  96788 72284 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-1]
			naas     1571343  0.0  1.8  96244 71620 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-2]
			naas     1699895  0.0  1.9  98568 73844 ?        S    12:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-3]
			a.gordon 1744961  0.0  0.0   4728   652 pts/0    S+   16:15   0:00 grep --color=auto nginx
			root     3341320  0.0  0.0  33008  2948 ?        Ss   Mar22   0:00 nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf
			nginx    3341321  0.0  0.1  34196  5300 ?        S    Mar22   0:04 nginx: worker process
			nginx    3341322  0.0  0.1  34196  5984 ?        S    Mar22   9:43 nginx: worker process
			nginx    3341323  0.0  0.1  34004  4300 ?        S    Mar22   0:14 nginx: cache manager process
			nginx    3669442  1.0  1.0 124264 39140 ?        Sl   Apr15 363:57 amplify-agent`),
			expected: map[string]bool{
				"3341323": true,
			},
		},
		"no cache manager proc should yield empty map": {
			input: []byte(`naas     1571264  0.1  2.0  97604 80636 ?        S    00:00   1:02 [celeryd: w0@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w0.log --pidfile=/var/run/naas/queue-w0.pid --hostname=w0@test.server.com)
			naas     1571284  0.1  2.0  97676 80952 ?        S    00:00   1:01 [celeryd: w1@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w1.log --pidfile=/var/run/naas/queue-w1.pid --hostname=w1@test.server.com)
			naas     1571325  0.0  1.8  96788 72148 ?        S    00:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-1]
			naas     1571342  0.0  1.8  96788 72284 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-1]
			naas     1571343  0.0  1.8  96244 71620 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-2]
			naas     1699895  0.0  1.9  98568 73844 ?        S    12:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-3]
			a.gordon 1744961  0.0  0.0   4728   652 pts/0    S+   16:15   0:00 grep --color=auto nginx
			root     3341320  0.0  0.0  33008  2948 ?        Ss   Mar22   0:00 nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf
			nginx    3341321  0.0  0.1  34196  5300 ?        S    Mar22   0:04 nginx: worker process
			nginx    3341322  0.0  0.1  34196  5984 ?        S    Mar22   9:43 nginx: worker process
			nginx    3669442  1.0  1.0 124264 39140 ?        Sl   Apr15 363:57 amplify-agent`),
			expected: map[string]bool{},
		},
		"many cache procs should yield >1 results": {
			input: []byte(`naas     1571264  0.1  2.0  97604 80636 ?        S    00:00   1:02 [celeryd: w0@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w0.log --pidfile=/var/run/naas/queue-w0.pid --hostname=w0@test.server.com)
			naas     1571284  0.1  2.0  97676 80952 ?        S    00:00   1:01 [celeryd: w1@test.server.com:MainProcess] -active- (worker -A naas.queue.celery:queue --loglevel=DEBUG -Ofair --time-limit=300 --concurrency=2 --maxtasksperchild=10 --logfile=/var/log/naas/queue-celery-w1.log --pidfile=/var/run/naas/queue-w1.pid --hostname=w1@test.server.com)
			naas     1571325  0.0  1.8  96788 72148 ?        S    00:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-1]
			naas     1571342  0.0  1.8  96788 72284 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-1]
			naas     1571343  0.0  1.8  96244 71620 ?        S    00:00   0:00 [celeryd: w1@test.server.com:ForkPoolWorker-2]
			naas     1699895  0.0  1.9  98568 73844 ?        S    12:00   0:00 [celeryd: w0@test.server.com:ForkPoolWorker-3]
			a.gordon 1744961  0.0  0.0   4728   652 pts/0    S+   16:15   0:00 grep --color=auto nginx
			root     3341320  0.0  0.0  33008  2948 ?        Ss   Mar22   0:00 nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf
			nginx    3341321  0.0  0.1  34196  5300 ?        S    Mar22   0:04 nginx: worker process
			nginx    3341322  0.0  0.1  34196  5984 ?        S    Mar22   9:43 nginx: worker process
			nginx    3341323  0.0  0.1  34004  4300 ?        S    Mar22   0:14 nginx: cache manager process
			nginx    3341324  0.0  0.1  34004  4300 ?        S    Mar22   0:14 nginx: cache worker process
			nginx    3341325  0.0  0.1  34004  4300 ?        S    Mar22   0:14 nginx: cache manager process
			nginx    3669442  1.0  1.0 124264 39140 ?        Sl   Apr15 363:57 amplify-agent`),
			expected: map[string]bool{
				"3341323": true,
				"3341324": true,
				"3341325": true,
			},
		},
	}

	for desc, test := range tests {
		cacheProcsMap := getCacheWorkersFromPSOut(test.input)
		assert.Equal(t, test.expected, cacheProcsMap, desc)
	}
}
