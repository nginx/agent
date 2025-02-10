// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package nginxprocess

import (
	"errors"

	"github.com/shirou/gopsutil/v4/process"
)

// errNotAnNginxProcess is returned when querying a process that is not an NGINX process.
var errNotAnNginxProcess = errors.New("not a NGINX process")

// IsNotNginxErr returns true if this error is due to the process not being an NGINX process.
func IsNotNginxErr(err error) bool { return errors.Is(err, errNotAnNginxProcess) }

// IsNotRunningErr returns true if this error is due to the OS process no longer running.
func IsNotRunningErr(err error) bool { return errors.Is(err, process.ErrorProcessNotRunning) }
