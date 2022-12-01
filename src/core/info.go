/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

type Info struct {
	name    *string
	version *string
}

func NewInfo(name string, version string) *Info {
	info := new(Info)
	info.name = &name
	info.version = &version
	return info
}

func (info *Info) Name() string {
	return *info.name
}

func (info *Info) Version() string {
	return *info.version
}
