// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

type Process struct {
	Name    string
	Cmd     string
	Exe     string
	Status  string
	PID     int32
	PPID    int32
	Running bool
}
