/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	ps "github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

type NginxWorker struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
	prevStats map[string]*WorkerStats
	binary    core.NginxBinary
	cl        NginxWorkerCollector
	init      sync.Once
	logger    *MetricSourceLogger
}

// NewNginxWorker collects metrics about nginx child processes
func NewNginxWorker(baseDimensions *metrics.CommonDim,
	namespace string,
	binary core.NginxBinary,
	collector NginxWorkerCollector) *NginxWorker {
	return &NginxWorker{
		baseDimensions: baseDimensions,
		namedMetric:    &namedMetric{namespace: namespace},
		binary:         binary,
		prevStats:      map[string]*WorkerStats{},
		cl:             collector,
		logger:         NewMetricSourceLogger(),
	}
}

func (c *NginxWorker) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	var err error
	defer wg.Done()
	childProcs := c.binary.GetChildProcesses()
	c.init.Do(func() {
		for pid, children := range childProcs {
			c.prevStats[pid], err = c.cl.GetWorkerStats(children)
			if err != nil {
				c.logger.Log(fmt.Sprintf("Failed to retrieve nginx process metrics, %v", err))
				c.prevStats[pid] = nil
				return
			}
		}
	})

	for pid, children := range childProcs {
		stats, err := c.cl.GetWorkerStats(children)
		if err != nil {
			c.logger.Log(fmt.Sprintf("Failed to retrieve nginx process metrics, %v", err))
			return
		}

		if c.prevStats[pid] == nil {
			c.prevStats[pid] = stats
		}

		c.group = "workers"

		// gauges are computed via counter delta
		cpuUser := stats.Workers.CPUUser - c.prevStats[pid].Workers.CPUUser
		if stats.Workers.CPUUser < c.prevStats[pid].Workers.CPUUser {
			cpuUser = stats.Workers.CPUUser
		}

		cpuSystem := stats.Workers.CPUSystem - c.prevStats[pid].Workers.CPUSystem
		if stats.Workers.CPUSystem < c.prevStats[pid].Workers.CPUSystem {
			cpuSystem = stats.Workers.CPUSystem
		}

		memRss := stats.Workers.MemRss - c.prevStats[pid].Workers.MemRss
		if stats.Workers.MemRss < c.prevStats[pid].Workers.MemRss {
			memRss = stats.Workers.MemRss
		}

		memVms := stats.Workers.MemVms - c.prevStats[pid].Workers.MemVms
		if stats.Workers.MemVms < c.prevStats[pid].Workers.MemVms {
			memVms = stats.Workers.MemVms
		}

		KbsR := stats.Workers.KbsR - c.prevStats[pid].Workers.KbsR
		if stats.Workers.KbsR < c.prevStats[pid].Workers.KbsR {
			KbsR = stats.Workers.KbsR
		}

		KbsW := stats.Workers.KbsW - c.prevStats[pid].Workers.KbsW
		if stats.Workers.KbsW < c.prevStats[pid].Workers.KbsW {
			KbsW = stats.Workers.KbsW
		}

		simpleMetrics := c.convertSamplesToSimpleMetrics(map[string]float64{
			"count":         float64(stats.Workers.Count),
			"rlimit_nofile": float64(stats.Workers.RlimitNofile),
			"cpu.user":      float64(cpuUser),
			"cpu.system":    float64(cpuSystem),
			"cpu.total":     float64(cpuSystem + cpuUser),
			"fds_count":     float64(stats.Workers.FdsCount),
			"mem.vms":       float64(memVms),
			"mem.rss":       float64(memRss),
			"mem.rss_pct":   float64(stats.Workers.MemRssPct),
			"io.kbs_r":      float64(KbsR),
			"io.kbs_w":      float64(KbsW),
		})

		select {
		case <-ctx.Done():
		case m <- metrics.NewStatsEntity(c.baseDimensions.ToDimensions(), simpleMetrics):
		}

		c.prevStats[pid] = stats
	}
}

func (c *NginxWorker) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.baseDimensions = dimensions
}

func (c *NginxWorker) Stop() {
	log.Debugf("Stopping NginxWorker source for nginx id: %v", c.baseDimensions.NginxId)
}

// NginxWorkerClient allows you to fetch NGINX worker
// metrics from psutil via the PID of the master procs
// it implements an interface that we can mock for testing
type NginxWorkerClient struct {
	logger *MetricSourceLogger
}

// WorkerStats represents NGINX worker metrics
type WorkerStats struct {
	Workers *Workers
}

// Workers represents metrics related to child nginx processes
type Workers struct {
	Count        float64
	KbsR         float64
	KbsW         float64
	CPUSystem    float64
	CPUTotal     float64
	CPUUser      float64
	FdsCount     float64
	MemVms       float64
	MemRss       float64
	MemRssPct    float64
	RlimitNofile float64
}

type NginxWorkerCollector interface {
	GetWorkerStats(childProcs []*proto.NginxDetails) (*WorkerStats, error)
}

func NewNginxWorkerClient() NginxWorkerCollector {
	return &NginxWorkerClient{NewMetricSourceLogger()}
}

// GetWorkerStats fetches the nginx master & worker metrics from psutil.
func (client *NginxWorkerClient) GetWorkerStats(childProcs []*proto.NginxDetails) (*WorkerStats, error) {
	stats := &WorkerStats{
		Workers: &Workers{},
	}

	cacheProcs, err := getCacheProcs()
	if err != nil {
		return stats, err
	}

	var numWorkers int
	var usr, sys, fdSum float64 = 0, 0, 0
	var memRss, memVms, memPct float64 = 0, 0, 0
	var kbsr, kbsw float64 = 0, 0
	for _, nginxDetails := range childProcs {
		if cacheProcs[nginxDetails.ProcessId] {
			continue
		}
		numWorkers++

		pidAsInt, err := strconv.Atoi(nginxDetails.ProcessId)
		if err != nil {
			client.logger.Log(fmt.Sprintf("Failed to convert %s to int: %v", nginxDetails.ProcessId, err))
			continue
		}

		proc, err := ps.NewProcess(int32(pidAsInt))
		if err != nil {
			client.logger.Log(fmt.Sprintf("Failed to retrieve process from pid %d: %v", pidAsInt, err))
			continue
		}

		if times, err := proc.Times(); err == nil {
			usr = usr + times.User
			sys = sys + times.System
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get CPU times metrics, %v", err))
		}

		if memstat, err := proc.MemoryInfo(); err == nil {
			memRss += float64(memstat.RSS)
			memVms += float64(memstat.VMS)
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get memory info metrics, %v", err))
		}

		if mempct, err := proc.MemoryPercent(); err == nil {
			memPct += float64(mempct)
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get memory percentage metrics, %v", err))
		}

		if fd, err := proc.NumFDs(); err == nil {
			fdSum = fdSum + float64(fd)
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get number of file descriptors used metrics, %v", err))
		}

		if rlimit, err := proc.Rlimit(); err == nil {
			var rlimitMax int32
			for _, rl := range rlimit {
				if rl.Resource == ps.RLIMIT_NOFILE && rl.Hard > rlimitMax {
					rlimitMax = rl.Hard
				}
			}
			stats.Workers.RlimitNofile = float64(rlimitMax)
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get resource limit metrics, %v", err))
		}

		if ioc, err := proc.IOCounters(); err == nil {
			kbsr += float64(ioc.ReadBytes / 1000)
			kbsw += float64(ioc.WriteBytes / 1000)
		} else {
			client.logger.Log(fmt.Sprintf("Unable to get io counter metrics, %v", err))
		}
	}

	stats.Workers.CPUUser = usr
	stats.Workers.CPUSystem = sys
	stats.Workers.CPUTotal = usr + sys
	stats.Workers.Count = float64(numWorkers)
	stats.Workers.MemRss = float64(memRss)
	stats.Workers.MemVms = float64(memVms)
	stats.Workers.MemRssPct = float64(memPct)
	stats.Workers.KbsR = kbsr
	stats.Workers.KbsW = kbsw
	stats.Workers.FdsCount = fdSum

	return stats, nil
}

// this method determines whether a child process is a cache
// management process for the nginx master. We exclude these from other
// child processes that we count as nginx workers
func getCacheProcs() (map[string]bool, error) {
	output, err := exec.Command("sh", "-c", "ps aux | grep nginx").Output()
	if err != nil {
		return map[string]bool{}, err
	}
	return getCacheWorkersFromPSOut(output), nil
}

func getCacheWorkersFromPSOut(b []byte) map[string]bool {
	cacheWorkers := map[string]bool{}
	outputSplit := strings.Split(string(b), "\n")
	for _, l := range outputSplit {
		if strings.Contains(l, "cache") {
			cols := strings.Fields(l)
			if len(cols) >= 2 {
				cacheWorkers[cols[1]] = true
			}
		}
	}
	return cacheWorkers
}
