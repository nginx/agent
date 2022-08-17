package core

import (
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// CheckForProcesses takes in a slice of strings that represents the process
// names to check for then returns a slice of strings of the processes that
// were checked for and NOT found.
func CheckForProcesses(processesToCheck []string) ([]string, error) {
	runningProcesses, err := process.Processes()
	if err != nil {
		return nil, err
	}

	processCheckCopy := make([]string, len(processesToCheck))
	copy(processCheckCopy, processesToCheck)

	for _, process := range runningProcesses {
		if len(processCheckCopy) == 0 {
			return processCheckCopy, nil
		}

		procName, err := process.Name()
		if err != nil {
			continue
		}

		procCmd, err := process.CmdlineSlice()
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
