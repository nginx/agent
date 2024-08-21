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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

// FileWatcher listens for data plane changes
type FileWatcher struct {
	messagePipeline core.MessagePipeInterface
	config          *config.Config
	watching        *sync.Map
	watcher         *fsnotify.Watcher
	wg              sync.WaitGroup
	ctx             context.Context
	env             core.Environment
	enabled         bool
	cancelFunction  context.CancelFunc
}

var emptyEvent = fsnotify.Event{
	Name: "",
	Op:   0,
}

const (
	Create = fsnotify.Create
	Write  = fsnotify.Write
	Remove = fsnotify.Remove
	Rename = fsnotify.Rename
	Chmod  = fsnotify.Chmod
)

func NewFileWatcher(config *config.Config, env core.Environment) *FileWatcher {

	fw := &FileWatcher{
		config:   config,
		watching: &sync.Map{},
		wg:       sync.WaitGroup{},
		env:      env,
		enabled:  true,
	}

	return fw
}

func (fw *FileWatcher) Init(pipeline core.MessagePipeInterface) {
	log.Info("FileWatcher initializing")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Error creating file watcher: %v", err)
		return
	}

	fw.watcher = watcher

	fw.messagePipeline = pipeline
	fw.ctx, fw.cancelFunction = context.WithCancel(fw.messagePipeline.Context())
	for dir := range fw.config.AllowedDirectoriesMap {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			log.Debugf("Skipping watching %s: %v", dir, err)
			continue
		}

		log.Debugf("Creating watcher for %v", dir)

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			_ = fw.addWatcher(info, path)
			return nil
		})
		if err != nil {
			log.Errorf("Error occurred creating watcher for %v: %v", dir, err)
		}
	}

	go fw.watchLoop()
}

func (fw *FileWatcher) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureFileWatcher, "v0.0.1")
}

func (fw *FileWatcher) Close() {
	log.Info("File Watcher is wrapping up")
	fw.enabled = false
	fw.cancelFunction()
	fw.watcher.Close()
}

func (fw *FileWatcher) Process(message *core.Message) {
	log.Debugf("File Watcher processing message: %v", message)
	switch message.Topic() {
	case core.FileWatcherEnabled:
		fw.enabled = message.Data().(bool)
		log.Debugf("File Watcher enabled: %v", fw.enabled)

		// If the file watcher is re-enabled we want to do a sync again
		// in case other files were modified on the system while it was disabled
		if fw.enabled {
			fw.messagePipeline.Process(core.NewMessage(core.DataplaneFilesChanged, nil))
		}
	}
}

func (fw *FileWatcher) Subscriptions() []string {
	return []string{
		core.FileWatcherEnabled,
	}
}

func (fw *FileWatcher) addWatcher(info os.FileInfo, path string) (err error) {
	fw.wg.Add(1)
	defer fw.wg.Done()
	if info == nil {
		info, err = fw.env.FileStat(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Debugf("Unable to add file watcher for %v as file doesn't exist: %v", path, err)
				return nil
			}
			log.Warnf("Error unable to add file watcher for %v : %v", path, err)
			return err
		}
	}

	if info.IsDir() && !fw.isWatching(path) {
		if err = fw.watcher.Add(path); err != nil {
			log.Errorf("Error occurred adding watcher for %s: %v", path, err)
			err := fw.watcher.Remove(path)
			if err != nil {
				log.Errorf("Error occurred removing watcher for %s: %v", path, err)
			}
			return err
		}
		fw.watching.Store(path, true)
	}
	return nil
}

func (fw *FileWatcher) removeWatcher(name string) error {
	fw.wg.Add(1)
	defer fw.wg.Done()
	if _, ok := fw.watching.Load(name); ok {
		err := fw.watcher.Remove(name)
		if err != nil {
			return err
		}

		fw.watching.Delete(name)
	}
	return nil
}

func (fw *FileWatcher) isWatching(name string) bool {
	v, _ := fw.watching.LoadOrStore(name, false)
	return v.(bool)
}

func (fw *FileWatcher) checkFailedWatch() {
	fw.watching.Range(func(key, value interface{}) bool {
		if !value.(bool) {
			_ = fw.addWatcher(nil, key.(string))
		}
		return true
	})
}

func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case <-fw.ctx.Done():
			return
		case event := <-fw.watcher.Events:

			if fw.enabled {
				if event == emptyEvent ||
					event.Name == "" ||
					strings.HasSuffix(event.Name, ".swp") ||
					strings.HasSuffix(event.Name, ".swx") ||
					strings.HasSuffix(event.Name, "~") {
					log.Tracef("Skipping FSNotify EVENT! %v\n", event)
					continue
				}

				switch {
				case event.Op&Write == Write:
					// We want to send messages on write since that means the contents changed,
					// but we already have a watcher on the file so nothing special needs to happen here
				case event.Op&Create == Create:
					err := fw.addWatcher(nil, event.Name)
					if err != nil {
						log.Errorf("Error occurred adding watcher for %v: %v", event.Name, err)
					}
				case event.Op&Remove == Remove, event.Op&Rename == Rename:
					err := fw.removeWatcher(event.Name)
					if err != nil {
						log.Errorf("Error occurred removing watcher for %v: %v", event.Name, err)
					}
				default:
					// We want to skip sending messages if it is not a write, create, or remove event.
					log.Tracef("DEFAULT %s", event.Op.String())
					continue
				}

				log.Tracef("Processing FSNotify EVENT! %v\n", event)

				fw.messagePipeline.Process(core.NewMessage(core.DataplaneFilesChanged, nil))
			}

		// watch for errors
		case err := <-fw.watcher.Errors:
			if err != nil {
				log.Errorf("ERROR %v", err)
			}
		case <-time.After(30 * time.Second):
			fw.checkFailedWatch()
		}
	}
}
