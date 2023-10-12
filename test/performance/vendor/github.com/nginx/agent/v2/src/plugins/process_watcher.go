/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

// ProcessWatcher listens for changes to nginx processes on the data plane
type ProcessWatcher struct {
	messagePipeline core.MessagePipeInterface
	ticker          *time.Ticker
	seenMasterProcs map[int32]*core.Process
	seenWorkerProcs map[int32]*core.Process
	nginxDetails    map[int32]*proto.NginxDetails
	wg              sync.WaitGroup
	env             core.Environment
	binary          core.NginxBinary
	processes       []*core.Process
	config          *config.Config
}

func NewProcessWatcher(env core.Environment, nginxBinary core.NginxBinary, processes []*core.Process, config *config.Config) *ProcessWatcher {
	return &ProcessWatcher{
		ticker:          time.NewTicker(5 * time.Second),
		seenMasterProcs: make(map[int32]*core.Process),
		seenWorkerProcs: make(map[int32]*core.Process),
		nginxDetails:    make(map[int32]*proto.NginxDetails),
		wg:              sync.WaitGroup{},
		env:             env,
		binary:          nginxBinary,
		processes:       processes,
		config:          config,
	}
}

func (pw *ProcessWatcher) Init(pipeline core.MessagePipeInterface) {
	log.Info("ProcessWatcher initializing")
	pw.messagePipeline = pipeline

	for _, proc := range pw.processes {
		if proc.IsMaster {
			pw.seenMasterProcs[proc.Pid] = proc
		} else {
			pw.seenWorkerProcs[proc.Pid] = proc
		}
		pw.nginxDetails[proc.Pid] = pw.binary.GetNginxDetailsFromProcess(proc)
	}

	pw.wg.Add(1)
	go pw.watchProcLoop(pipeline.Context())

	pw.messagePipeline.Process(core.NewMessage(core.NginxDetailProcUpdate, pw.processes))
}

func (pw *ProcessWatcher) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureProcessWatcher, "v0.0.1")
}

func (pw *ProcessWatcher) Close() {
	pw.ticker.Stop()
	pw.seenMasterProcs = nil
	pw.seenWorkerProcs = nil
	pw.nginxDetails = nil

	log.Info("ProcessWatcher is wrapping up")
}

func (pw *ProcessWatcher) Process(message *core.Message) {}

func (pw *ProcessWatcher) Subscriptions() []string {
	return []string{}
}

func (pw *ProcessWatcher) watchProcLoop(ctx context.Context) {
	defer pw.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pw.ticker.C:
			nginxProcs := pw.env.Processes()
			procUpdates, runningMasterProcs, runningWorkerProcs := pw.getProcUpdates(nginxProcs)
			if len(procUpdates) > 0 {
				pw.messagePipeline.Process(procUpdates...)
				pw.seenMasterProcs = runningMasterProcs
				pw.seenWorkerProcs = runningWorkerProcs

				pw.messagePipeline.Process(core.NewMessage(core.NginxDetailProcUpdate, nginxProcs))
			}

			if len(pw.seenWorkerProcs) > pw.config.QueueSize {
				log.Warnf(
					"Number of NGINX worker processes (%d) is greater than queue size (%d). Update NGINX Agent config to increase the queue size so that it is greater than the number of NGINX worker processes.",
					len(pw.seenWorkerProcs),
					pw.config.QueueSize,
				)
			}
		}
	}
}

// getProcUpdates returns a slice of updates to process, currently running master proc map, currently running worker process map
func (pw *ProcessWatcher) getProcUpdates(nginxProcs []*core.Process) ([]*core.Message, map[int32]*core.Process, map[int32]*core.Process) {
	procUpdates := []*core.Message{}

	// create maps of currently running processes
	runningMasterProcs := make(map[int32]*core.Process)
	runningWorkerProcs := make(map[int32]*core.Process)
	for _, proc := range nginxProcs {
		if proc.IsMaster {
			runningMasterProcs[proc.Pid] = proc
		} else {
			runningWorkerProcs[proc.Pid] = proc
		}
	}

	// send messages for new processes that were created
	for _, proc := range nginxProcs {
		if _, ok := pw.seenMasterProcs[proc.Pid]; ok {
			continue
		}
		if _, ok := pw.seenWorkerProcs[proc.Pid]; ok {
			continue
		}

		pw.nginxDetails[proc.Pid] = pw.binary.GetNginxDetailsFromProcess(proc)
		if proc.IsMaster {
			log.Debugf("Processing process change event: new master proc %d", proc.Pid)
			procUpdates = append(procUpdates, core.NewMessage(core.NginxMasterProcCreated, pw.nginxDetails[proc.Pid]))
		} else {
			log.Debugf("Processing process change event: new worker proc %d", proc.Pid)
			procUpdates = append(procUpdates, core.NewMessage(core.NginxWorkerProcCreated, pw.nginxDetails[proc.Pid]))
		}
	}

	// send messages for old master processes that have been killed
	for pid, proc := range pw.seenMasterProcs {
		if _, ok := runningMasterProcs[pid]; !ok {
			log.Debugf("Processing process change event: old master proc killed %d", pid)
			procUpdates = append(procUpdates, core.NewMessage(core.NginxMasterProcKilled, pw.nginxDetails[proc.Pid]))
			delete(pw.nginxDetails, proc.Pid)
		}
	}

	// send messages for old worker processes that have been killed
	for pid, proc := range pw.seenWorkerProcs {
		if _, ok := runningWorkerProcs[pid]; !ok {
			log.Debugf("Processing process change event: old worker proc killed %d", pid)
			procUpdates = append(procUpdates, core.NewMessage(core.NginxWorkerProcKilled, pw.nginxDetails[proc.Pid]))
			delete(pw.nginxDetails, proc.Pid)
		}
	}

	return procUpdates, runningMasterProcs, runningWorkerProcs
}
