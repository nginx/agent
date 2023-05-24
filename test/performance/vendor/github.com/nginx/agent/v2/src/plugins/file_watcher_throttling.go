/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/v2/src/core"
)

const (
	Duration          = 2 * time.Second
	InitialInterval   = 100 * time.Millisecond
	MaxInterval       = 500 * time.Millisecond
	MaxElapsedTimeout = 10 * time.Second
)

type FileWatchThrottle struct {
	messagePipeline core.MessagePipeInterface
	startedMutex    sync.RWMutex
	started         bool
	canSend         *atomic.Bool
	last            atomic.Time
	duration        time.Duration
}

func NewFileWatchThrottle() *FileWatchThrottle {
	return &FileWatchThrottle{
		canSend:  atomic.NewBool(false),
		started:  false,
		last:     *atomic.NewTime(time.Now()),
		duration: Duration,
	}
}

func (fwt *FileWatchThrottle) GetStarted() bool {
	fwt.startedMutex.RLock()
	defer fwt.startedMutex.RUnlock()
	return fwt.started
}

func (fwt *FileWatchThrottle) SetStarted(newValue bool) {
	fwt.startedMutex.Lock()
	fwt.started = newValue
	defer fwt.startedMutex.Unlock()
}

func (fwt *FileWatchThrottle) Init(pipeline core.MessagePipeInterface) {
	fwt.messagePipeline = pipeline
	log.Info("FileWatchThrottle initializing")
}

func (fwt *FileWatchThrottle) Close() {
	log.Info("FileWatchThrottle is wrapping up")
}

func (fwt *FileWatchThrottle) Info() *core.Info {
	return core.NewInfo("File Watch Throttle", "v0.0.1")
}

func (fwt *FileWatchThrottle) Process(msg *core.Message) {
	if msg.Exact(core.DataplaneFilesChanged) {
		log.Tracef("started DataplaneFilesChanged processing %v", fwt.GetStarted())
		fwt.last.Store(time.Now().Add(fwt.duration))
		if !fwt.GetStarted() {
			fwt.SetStarted(true)
			go fwt.waitUntilNoMoreSignals()
		}
	}
}

func (fwt *FileWatchThrottle) Subscriptions() []string {
	return []string{core.DataplaneFilesChanged}
}

func (fwt *FileWatchThrottle) waitUntilNoMoreSignals() {
	backoffSetting := backoff.BackoffSettings{
		InitialInterval: InitialInterval,
		MaxInterval:     MaxInterval,
		MaxElapsedTime:  MaxElapsedTimeout,
		Jitter:          backoff.BACKOFF_JITTER,
		Multiplier:      backoff.BACKOFF_MULTIPLIER,
	}
	err := backoff.WaitUntil(context.Background(), backoffSetting, fwt.retry)
	if err != nil {
		log.Warnf("Warring, issue occurred waiting until there were no more signals %v", err)
	}

	if fwt.canSend.Load() {
		fwt.canSend.Store(false)
		fwt.messagePipeline.Process(core.NewMessage(core.DataplaneChanged, nil))
	}
	fwt.SetStarted(false)
}

func (fwt *FileWatchThrottle) retry() error {
	since := time.Since(fwt.last.Load())
	if since.Milliseconds() >= 0 {
		fwt.canSend.Store(true)
		return nil
	}
	return errors.New("retry")
}
