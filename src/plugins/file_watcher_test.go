/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var directories = []string{
	"watching",
	"watching/subdir1",
	"watching/subdir2",
}

var files = []string{
	"nginx.conf",
	"index.txt",
	"default.json",
}

type testObj struct {
	name                 string
	expectedMessageCount int
	tmpDir               string
	dirs                 []string
	files                []string
}

const (
	Milliseconds = 150
)

func TestWatcherCreatingSubDirectories(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 2,
		},
	}
	for _, test := range tests {

		env := tutils.MockEnvironment{}
		env.Mock.On("FileStat", mock.Anything).Return(env.FileStat)

		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDir(tt, test.tmpDir, directories[0]))

			file := path.Join(test.dirs[0], files[0])
			writeFile(tt, file)
			test.files = append(test.files, file)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &env)

			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			test.dirs = append(test.dirs, writeDir(tt, test.tmpDir, directories[1]))
			test.dirs = append(test.dirs, writeDir(tt, test.tmpDir, directories[2]))
			defer removeDirs(tt, test.dirs)

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()

			pluginUnderTest.Close()
		})
	}
}

func TestWatcherMovingDirectories(t *testing.T) {
	tests := []testObj{
		{
			name: "test watcher",
			// expect 2 creates and 2 remove|renames
			expectedMessageCount: 4,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			env := tutils.MockEnvironment{}
			env.Mock.On("FileStat", mock.Anything).Return(env.FileStat)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &env)
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			moveFileOrDir(tt, test.dirs[1], test.dirs[1]+"Seconds")
			moveFileOrDir(tt, test.dirs[2], test.dirs[2]+"4")

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()

			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherDeletingDirectories(t *testing.T) {
	tests := []testObj{
		{
			name: "test watcher",
			// expectation is 4 messages as removed 2 dirs and 2 files
			expectedMessageCount: 4,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &tutils.MockEnvironment{})
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			removeDir(tt, test.dirs[1])
			removeDir(tt, test.dirs[2])

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherFixingPermissions(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 1,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &tutils.MockEnvironment{})
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			changePermissions(t, 0111, test.dirs[1])

			err := os.Chmod(test.dirs[1], 0111)
			assert.NoError(t, err)

			time.Sleep(Milliseconds * time.Millisecond)
			updateFile(tt, test.files[1], "new content 1")
			time.Sleep(5 * Milliseconds * time.Millisecond)

			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()

			// resetting permissions after the test so can teardown
			changePermissions(t, 0777, test.dirs[1])
		})
	}
}

func TestWatcherCreatingFiles(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 2,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)
			writeFile(tt, path.Join(test.dirs[0], files[0]))

			env := tutils.MockEnvironment{}
			env.Mock.On("FileStat", mock.Anything).Return(env.FileStat)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &env)
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)
			writeFile(tt, path.Join(test.dirs[1], files[1]))
			writeFile(tt, path.Join(test.dirs[2], files[2]))

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherCreatingFilesMultiple(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 3,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)
			writeFile(tt, path.Join(test.dirs[0], files[0]))

			env := tutils.MockEnvironment{}
			env.Mock.On("FileStat", mock.Anything).Return(env.FileStat)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &env)
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)
			writeFile(tt, path.Join(test.dirs[1], files[1]))
			writeFile(tt, path.Join(test.dirs[2], files[2]))

			time.Sleep(Milliseconds * time.Millisecond)

			writeFile(tt, path.Join(test.dirs[2], files[1]))
			writeFile(tt, path.Join(test.dirs[2], files[1]))

			time.Sleep(Milliseconds * time.Millisecond)

			messagePipe.RunWithoutInit()
			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
			defer removeDirs(tt, test.dirs)
		})
	}
}

func TestWatcherMovingFiles(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 4,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			env := tutils.MockEnvironment{}
			env.Mock.On("FileStat", mock.Anything).Return(env.FileStat)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &env)
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			moveFileOrDir(tt, test.files[1], path.Join(path.Dir(test.files[2]), files[1]))
			moveFileOrDir(tt, test.files[2], path.Join(path.Dir(test.files[1]), files[2]))

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherUpdatingFiles(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 2,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &tutils.MockEnvironment{})
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			updateFile(tt, test.files[1], "new content 1")
			updateFile(tt, test.files[2], "new content 2")

			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherDeleteFiles(t *testing.T) {
	tests := []testObj{
		{
			name:                 "test watcher",
			expectedMessageCount: 2,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			test.tmpDir = tt.TempDir()
			test.dirs = append(test.dirs, writeDirs(tt, test.tmpDir)...)
			test.files = append(test.files, writeFiles(tt, test.dirs)...)

			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{test.dirs[0]: {}}}, &tutils.MockEnvironment{})
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			time.Sleep(Milliseconds * time.Millisecond)

			deleteFile(tt, test.files[0])
			deleteFile(tt, test.files[2])
			time.Sleep(Milliseconds * time.Millisecond)
			messagePipe.RunWithoutInit()
			defer removeDirs(tt, test.dirs)

			result := messagePipe.GetProcessedMessages()

			assert.GreaterOrEqual(tt, len(result), test.expectedMessageCount, result)

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func TestWatcherProcess(t *testing.T) {
	tests := []struct {
		name           string
		messagesToSend []*core.Message
		messageTopics  []string
	}{
		{
			name: "test disabling file watcher",
			messagesToSend: []*core.Message{
				core.NewMessage(core.FileWatcherEnabled, false),
			},
			messageTopics: []string{
				core.FileWatcherEnabled,
			},
		}, {
			name: "test re-enabling file watcher",
			messagesToSend: []*core.Message{
				core.NewMessage(core.FileWatcherEnabled, false),
				core.NewMessage(core.FileWatcherEnabled, true),
			},
			messageTopics: []string{
				core.FileWatcherEnabled,
				core.FileWatcherEnabled,
				core.DataplaneFilesChanged,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			pluginUnderTest := NewFileWatcher(&config.Config{AllowedDirectoriesMap: map[string]struct{}{"/tmp": {}}}, &tutils.MockEnvironment{})
			ctx, cancelCTX := context.WithCancel(context.Background())
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			pluginUnderTest.Init(messagePipe)

			for _, message := range test.messagesToSend {
				messagePipe.Process(message)
			}
			messagePipe.RunWithoutInit()

			processedMessages := messagePipe.GetProcessedMessages()
			if len(processedMessages) != len(test.messageTopics) {
				tt.Fatalf("expected %d messages, received %d", len(test.messageTopics), len(processedMessages))
			}
			for idx, msg := range processedMessages {
				if test.messageTopics[idx] != msg.Topic() {
					tt.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), test.messageTopics[idx])
				}
			}

			defer cancelCTX()
			pluginUnderTest.Close()
		})
	}
}

func writeFile(t *testing.T, file string) {
	err := ioutil.WriteFile(file, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func writeFiles(t *testing.T, dirs []string) (ret []string) {
	if len(dirs) != len(files) {
		t.Fatal("number of directories must equal the number of files")
	}
	ret = make([]string, len(files))
	for idx, file := range files {
		ret[idx] = path.Join(dirs[idx], file)
		writeFile(t, ret[idx])
	}

	return ret
}

func updateFile(t *testing.T, file, content string) {
	err := ioutil.WriteFile(file, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to update file: %v", err)
	}
}

func deleteFile(t *testing.T, location string) {
	err := os.Remove(location)
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}
}

func moveFileOrDir(t *testing.T, oldLocation, newLocation string) {
	err := os.Rename(oldLocation, newLocation)
	if err != nil {
		t.Fatalf("failed to move file: %v", err)
	}
}

func writeDir(t *testing.T, base, directory string) string {
	dir := path.Join(base, directory)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	return dir
}

func writeDirs(t *testing.T, base string) (dirs []string) {
	for _, dir := range directories {
		dirs = append(dirs, writeDir(t, base, dir))
	}

	return dirs
}

func removeDir(t *testing.T, location string) {
	err := os.RemoveAll(location)
	if err != nil {
		t.Fatalf("failed to delete directory: %v", err)
	}
}

func removeDirs(t *testing.T, dirs []string) {
	for _, dir := range dirs {
		removeDir(t, dir)
	}
}

func changePermissions(t *testing.T, fileMode uint32, directory string) {
	err := os.Chmod(directory, os.FileMode(fileMode))
	if err != nil {
		t.Fail()
	}
}
