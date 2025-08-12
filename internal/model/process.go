// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

type ProcessInfo struct {
	ConfigureArgs   map[string]interface{}
	Version         string
	Prefix          string
	ConfPath        string
	ExePath         string
	LoadableModules []string
	DynamicModules  []string
	ProcessID       int32
}
