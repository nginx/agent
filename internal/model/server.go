// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

type ServerType int

const (
	Command ServerType = iota
	Auxiliary
)

var serverType = map[ServerType]string{
	Command:   "command",
	Auxiliary: "auxiliary",
}

func (s ServerType) String() string {
	return serverType[s]
}
