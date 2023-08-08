/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"context"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// CheckForProcesses takes in a slice of strings that represents the process
// names to check for then returns a slice of strings of the processes that
// were checked for and NOT found.
func CheckForProcesses(processesToCheck []string) ([]string, error) {
	ctx := context.Background()
	defer ctx.Done()

	runningProcesses, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	processCheckCopy := make([]string, len(processesToCheck))
	copy(processCheckCopy, processesToCheck)

	for _, process := range runningProcesses {
		if len(processCheckCopy) == 0 {
			return processCheckCopy, nil
		}

		procName, err := process.NameWithContext(ctx)
		if err != nil {
			continue
		}

		procCmd, err := process.CmdlineSliceWithContext(ctx)
		if err != nil {
			continue
		}

		if found, idx := SliceContainsString(processCheckCopy, procName); found {
			processCheckCopy = append(processCheckCopy[:idx], processCheckCopy[idx+1:]...)
		} else if len(procCmd) > 0 {
			splitCmd := strings.Split(procCmd[0], "/")
			procName = splitCmd[len(splitCmd)-1]
			if found, idx := SliceContainsString(processCheckCopy, procName); found {
				processCheckCopy = append(processCheckCopy[:idx], processCheckCopy[idx+1:]...)
			}
		}
	}

	return processCheckCopy, nil
}
